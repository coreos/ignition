// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The azure provider fetches a configuration from the Azure OVF DVD.

package azure

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/coreos/ignition/v2/config/shared/errors"
	cfgutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	execUtil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/vincent-petithory/dataurl"

	"github.com/coreos/vcontext/report"
	"golang.org/x/sys/unix"
)

const (
	configPath = "/CustomData.bin"
)

// These constants come from <cdrom.h>.
const (
	CDROM_DRIVE_STATUS = 0x5326
)

// These constants come from <cdrom.h>.
const (
	CDS_NO_INFO = iota
	CDS_NO_DISC
	CDS_TRAY_OPEN
	CDS_DRIVE_NOT_READY
	CDS_DISC_OK
)

// Azure uses a UDF volume for the OVF configuration.
const (
	CDS_FSTYPE_UDF = "udf"
)

var (
	imdsUserdataURL = url.URL{
		Scheme:   "http",
		Host:     "169.254.169.254",
		Path:     "metadata/instance/compute/userData",
		RawQuery: "api-version=2021-01-01&format=text",
	}
	imdsInstanceURL = url.URL{
		Scheme:   "http",
		Host:     "169.254.169.254",
		Path:     "metadata/instance",
		RawQuery: "api-version=2021-01-01&format=json&extended=true",
	}
)

var imdsRetryCodes = []int{
	404,
	410,
	429,
}

func init() {
	platform.Register(platform.Provider{
		Name:                "azure",
		NewFetcher:          newFetcher,
		Fetch:               fetchConfig,
		GenerateCloudConfig: generateCloudConfig,
	})
}

// fetchConfig wraps fetchFromAzureMetadata to implement the provider fetch interface.
func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	return fetchFromAzureMetadata(f)
}

// newFetcher returns a fetcher that tries to authenticate with Azure's default credential chain.
func newFetcher(l *log.Logger) (resource.Fetcher, error) {
	// Read about NewDefaultAzureCredential https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential
	// DefaultAzureCredential is a default credential chain for applications deployed to azure.
	session, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		l.Info("could not retrieve any azure credentials: %v", err)
		session = nil
	}

	return resource.Fetcher{
		Logger:    l,
		AzSession: session,
	}, nil
}

// fetchFromAzureMetadata first tries to fetch userData from IMDS then fallback on customData in case
// of empty config.
func fetchFromAzureMetadata(f *resource.Fetcher) (types.Config, report.Report, error) {
	// fetch-offline is not supported since we first try to get config from Azure IMDS.
	// this config fetching can only happen during fetch stage.
	if f.Offline {
		return types.Config{}, report.Report{}, resource.ErrNeedNet
	}

	logger := f.Logger

	// we first try to fetch config from IMDS, in case of failure we fallback on the custom data.
	userData, err := fetchFromIMDS(f)
	if err == nil {
		logger.Info("config has been read from IMDS userdata")
		return util.ParseConfig(logger, userData)
	}

	if err != errors.ErrEmpty {
		return types.Config{}, report.Report{}, err
	}

	logger.Debug("failed to retrieve userdata from IMDS, falling back to custom data: %v", err)
	return FetchFromOvfDevice(f, []string{CDS_FSTYPE_UDF})
}

// fetchFromIMDS requests the Azure IMDS to fetch userdata and decode it.
func fetchFromIMDS(f *resource.Fetcher) ([]byte, error) {
	headers := make(http.Header)
	headers.Set("Metadata", "true")

	// Azure IMDS expects some codes <500 to still be retried...
	// Here, we match the cloud-init set.
	// https://github.com/canonical/cloud-init/commit/c1a2047cf291
	// https://github.com/coreos/ignition/issues/1806
	data, err := f.FetchToBuffer(imdsUserdataURL, resource.FetchOptions{Headers: headers, RetryCodes: imdsRetryCodes})
	if err != nil {
		return nil, fmt.Errorf("fetching to buffer: %w", err)
	}

	n := len(data)

	if n == 0 {
		return nil, errors.ErrEmpty
	}

	// data is base64 encoded by the IMDS
	userData := make([]byte, base64.StdEncoding.DecodedLen(n))

	// we keep the number of bytes written to return [:l] only.
	// otherwise last byte will be 0x00 which makes fail the JSON's unmarshalling.
	l, err := base64.StdEncoding.Decode(userData, data)
	if err != nil {
		return nil, fmt.Errorf("decoding userdata: %w", err)
	}

	return userData[:l], nil
}

// FetchFromOvfDevice has the NewFetcher return signature. It is
// wrapped by this and AzureStack packages.
func FetchFromOvfDevice(f *resource.Fetcher, ovfFsTypes []string) (types.Config, report.Report, error) {
	logger := f.Logger
	checkedDevices := make(map[string]struct{})
	for {
		for _, ovfFsType := range ovfFsTypes {
			devices, err := execUtil.GetBlockDevices(ovfFsType)
			if err != nil {
				return types.Config{}, report.Report{}, fmt.Errorf("failed to retrieve block devices with FSTYPE=%q: %v", ovfFsType, err)
			}
			for _, dev := range devices {
				_, checked := checkedDevices[dev]
				// verify that this is a CD-ROM drive. This helps
				// to avoid reading data from an arbitrary block
				// device attached to the VM by the user.
				if !checked && isCdromPresent(logger, dev) {
					rawConfig, err := getRawConfig(f, dev, ovfFsType)
					if err != nil {
						logger.Debug("failed to retrieve config from device %q: %v", dev, err)
					} else {
						logger.Info("config has been read from custom data")
						return util.ParseConfig(logger, rawConfig)
					}
				}
				checkedDevices[dev] = struct{}{}
			}
		}
		// wait for the actual config drive to appear
		// if it's not shown up yet
		time.Sleep(time.Second)
	}
}

// getRawConfig returns the config by mounting the given block device
func getRawConfig(f *resource.Fetcher, devicePath string, fstype string) ([]byte, error) {
	logger := f.Logger
	logger.Debug("reading config")
	return readFileFromDevice(f, devicePath, fstype, configPath)
}

// isCdromPresent verifies if the given config drive is CD-ROM
func isCdromPresent(logger *log.Logger, devicePath string) bool {
	logger.Debug("opening config device: %q", devicePath)
	device, err := os.Open(devicePath)
	if err != nil {
		logger.Info("failed to open config device: %v", err)
		return false
	}
	defer func() {
		_ = device.Close()
	}()

	logger.Debug("getting drive status for %q", devicePath)
	status, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(device.Fd()),
		uintptr(CDROM_DRIVE_STATUS),
		uintptr(0),
	)

	switch status {
	case CDS_NO_INFO:
		logger.Info("drive status: no info")
	case CDS_NO_DISC:
		logger.Info("drive status: no disc")
	case CDS_TRAY_OPEN:
		logger.Info("drive status: open")
	case CDS_DRIVE_NOT_READY:
		logger.Info("drive status: not ready")
	case CDS_DISC_OK:
		logger.Info("drive status: OK")
	default:
		logger.Err("failed to get drive status: %s", errno.Error())
	}

	return (status == CDS_DISC_OK)
}

func readFileFromDevice(f *resource.Fetcher, devicePath string, fstype string, relativePath string) ([]byte, error) {
	logger := f.Logger
	mnt, err := os.MkdirTemp("", "ignition-azure")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(mnt); removeErr != nil {
			logger.Warning("failed to remove temp directory %q: %v", mnt, removeErr)
		}
	}()

	logger.Debug("mounting config device")
	if err := logger.LogOp(
		func() error { return unix.Mount(devicePath, mnt, fstype, unix.MS_RDONLY, "") },
		"mounting %q at %q", devicePath, mnt,
	); err != nil {
		return nil, fmt.Errorf("failed to mount device %q at %q: %v", devicePath, mnt, err)
	}
	defer func() {
		_ = logger.LogOp(
			func() error { return unix.Unmount(mnt, 0) },
			"unmounting %q at %q", devicePath, mnt,
		)
	}()

	logger.Debug("checking for config drive")
	if _, err := os.Stat(filepath.Join(mnt, "ovf-env.xml")); err != nil {
		return nil, fmt.Errorf("device %q does not appear to be a config drive: %v", devicePath, err)
	}

	target := filepath.Join(mnt, strings.TrimPrefix(relativePath, "/"))
	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %q from device %q: %v", relativePath, devicePath, err)
	}
	return data, nil
}

type instanceMetadata struct {
	Compute instanceComputeMetadata `json:"compute"`
}

type instanceComputeMetadata struct {
	Hostname   string              `json:"hostname"`
	OSProfile  instanceOSProfile   `json:"osProfile"`
	PublicKeys []instancePublicKey `json:"publicKeys"`
}

type instanceOSProfile struct {
	AdminUsername string `json:"adminUsername"`
}

type instancePublicKey struct {
	KeyData string `json:"keyData"`
}

type provisioningEnvelope struct {
	LinuxProvisioningConfigurationSet linuxProvisioningConfigurationSet `xml:"LinuxProvisioningConfigurationSet"`
}

type linuxProvisioningConfigurationSet struct {
	HostName                         string     `xml:"HostName"`
	UserName                         string     `xml:"UserName"`
	UserPassword                     string     `xml:"UserPassword"`
	DisableSshPasswordAuthentication string     `xml:"DisableSshPasswordAuthentication"`
	SSH                              sshSection `xml:"SSH"`
	CustomData                       string     `xml:"CustomData"`
	UserData                         string     `xml:"UserData"`
}

type sshSection struct {
	PublicKeys []sshPublicKey `xml:"PublicKeys>PublicKey"`
}

type sshPublicKey struct {
	Value string `xml:"Value"`
}

func (l linuxProvisioningConfigurationSet) passwordAuthDisabled() bool {
	disabled, err := strconv.ParseBool(strings.TrimSpace(l.DisableSshPasswordAuthentication))
	if err != nil {
		return false
	}
	return disabled
}

func generateCloudConfig(f *resource.Fetcher) (types.Config, error) {
	meta, err := fetchInstanceMetadata(f)
	if err != nil {
		return types.Config{}, fmt.Errorf("fetching instance metadata: %w", err)
	}

	ovfRaw, err := readOvfEnvironment(f, []string{CDS_FSTYPE_UDF})
	if err != nil {
		return types.Config{}, fmt.Errorf("reading provisioning metadata: %w", err)
	}
	if len(ovfRaw) == 0 {
		return types.Config{}, fmt.Errorf("ovf-env.xml was empty")
	}

	provisioning, err := parseProvisioningConfig(ovfRaw)
	if err != nil {
		return types.Config{}, fmt.Errorf("parsing provisioning metadata: %w", err)
	}

	return buildGeneratedConfig(meta, provisioning)
}

func fetchInstanceMetadata(f *resource.Fetcher) (*instanceMetadata, error) {
	headers := make(http.Header)
	headers.Set("Metadata", "true")
	data, err := f.FetchToBuffer(imdsInstanceURL, resource.FetchOptions{Headers: headers, RetryCodes: imdsRetryCodes})
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}

	var meta instanceMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}
	return &meta, nil
}

func readOvfEnvironment(f *resource.Fetcher, ovfFsTypes []string) ([]byte, error) {
	logger := f.Logger
	checkedDevices := make(map[string]struct{})
	for {
		for _, ovfFsType := range ovfFsTypes {
			devices, err := execUtil.GetBlockDevices(ovfFsType)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve block devices with FSTYPE=%q: %v", ovfFsType, err)
			}
			for _, dev := range devices {
				if _, checked := checkedDevices[dev]; checked {
					continue
				}
				if isCdromPresent(logger, dev) {
					data, err := readFileFromDevice(f, dev, ovfFsType, "ovf-env.xml")
					if err != nil {
						logger.Debug("failed to read ovf environment from device %q: %v", dev, err)
					} else if len(data) > 0 {
						return data, nil
					}
				}
				checkedDevices[dev] = struct{}{}
			}
		}
		time.Sleep(time.Second)
	}
}

func parseProvisioningConfig(raw []byte) (*linuxProvisioningConfigurationSet, error) {
	var env provisioningEnvelope
	if err := xml.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	return &env.LinuxProvisioningConfigurationSet, nil
}

func buildGeneratedConfig(meta *instanceMetadata, provisioning *linuxProvisioningConfigurationSet) (types.Config, error) {
	username := strings.TrimSpace(meta.Compute.OSProfile.AdminUsername)
	if username == "" {
		username = strings.TrimSpace(provisioning.UserName)
	}
	if username == "" {
		return types.Config{}, fmt.Errorf("unable to determine admin username from metadata or provisioning data")
	}

	password := strings.TrimSpace(provisioning.UserPassword)
	passwordAuthDisabled := provisioning.passwordAuthDisabled()

	sshKeys := collectSSHPublicKeys(meta, provisioning)

	user := types.PasswdUser{
		Name:              username,
		Groups:            []types.Group{"wheel"},
		HomeDir:           cfgutil.StrToPtr(fmt.Sprintf("/home/%s", username)),
		Shell:             cfgutil.StrToPtr("/bin/bash"),
		SSHAuthorizedKeys: sshKeys,
	}
	if password != "" {
		user.PasswordHash = cfgutil.StrToPtr(password)
	}

	sudoersFile := newDataFile("/etc/sudoers.d/99_wheel_nopasswd", 0440, "%wheel ALL=(ALL) NOPASSWD:ALL\n")
	passwordSetting := "yes"
	if passwordAuthDisabled {
		passwordSetting = "no"
	}
	sshConfig := fmt.Sprintf(`# Custom SSHD settings
PasswordAuthentication %s
PermitRootLogin no
AllowUsers %s
`, passwordSetting, username)
	sshdFile := newDataFile("/etc/ssh/sshd_config.d/10-custom.conf", 0644, sshConfig)

	return types.Config{
		Ignition: types.Ignition{
			Version: types.MaxVersion.String(),
		},
		Passwd: types.Passwd{
			Users: []types.PasswdUser{user},
		},
		Storage: types.Storage{
			Files: []types.File{sudoersFile, sshdFile},
		},
	}, nil
}

func collectSSHPublicKeys(meta *instanceMetadata, provisioning *linuxProvisioningConfigurationSet) []types.SSHAuthorizedKey {
	seen := make(map[string]struct{})
	var keys []types.SSHAuthorizedKey

	if meta != nil {
		for _, k := range meta.Compute.PublicKeys {
			key := strings.TrimSpace(k.KeyData)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, types.SSHAuthorizedKey(key))
		}
	}

	if provisioning != nil {
		for _, pk := range provisioning.SSH.PublicKeys {
			key := strings.TrimSpace(pk.Value)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, types.SSHAuthorizedKey(key))
		}
	}

	return keys
}

func newDataFile(path string, mode int, contents string) types.File {
	encoded := dataurl.EncodeBytes([]byte(contents))
	return types.File{
		Node: types.Node{
			Path: path,
		},
		FileEmbedded1: types.FileEmbedded1{
			Mode:     cfgutil.IntToPtr(mode),
			Contents: types.Resource{Source: &encoded},
		},
	}
}
