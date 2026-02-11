// Copyright 2020 Red Hat
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

package disks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	execUtil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/vincent-petithory/dataurl"
)

var (
	ErrBadVolume = errors.New("volume is not of the correct type")
	cexRegx      = regexp.MustCompile(`[0-9a-f]{2}\.[0-9a-f]{4}`)
)

// https://github.com/latchset/clevis/blob/master/src/pins/tang/clevis-encrypt-tang.1.adoc#config
type Tang struct {
	URL           string `json:"url"`
	Thumbprint    string `json:"thp,omitempty"`
	Advertisement any    `json:"adv,omitempty"`
}

// https://github.com/latchset/clevis/blob/master/README.md#pin-shamir-secret-sharing
type Pin struct {
	Tpm  bool
	Tang []Tang
}

func (p Pin) MarshalJSON() ([]byte, error) {
	if p.Tpm {
		return json.Marshal(&struct {
			Tang []Tang   `json:"tang,omitempty"`
			Tpm  struct{} `json:"tpm2"`
		}{
			Tang: p.Tang,
			Tpm:  struct{}{},
		})
	} else {
		return json.Marshal(&struct {
			Tang []Tang `json:"tang"`
		}{
			Tang: p.Tang,
		})
	}
}

type Clevis struct {
	Pins      Pin `json:"pins"`
	Threshold int `json:"t"`
}

// Initially tested generating keyfiles via dd'ing to a file from /dev/urandom
// however while cryptsetup had no problem with these keyfiles clevis seemed to
// die on them while keyfiles generated via openssl rand -hex would work...
func randHex(length int) (string, error) {
	bytes := make([]byte, length)
	// On older kernels this could block indefinitely but there's nothing
	// that we can do about it; we don't want to use earlyrand
	// https://lwn.net/Articles/802360/
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *stage) createLuks(config types.Config) error {
	if len(config.Storage.Luks) == 0 {
		return nil
	}
	s.PushPrefix("createLuks")
	defer s.PopPrefix()

	devs := []string{}
	for _, luks := range config.Storage.Luks {
		devs = append(devs, *luks.Device)
	}

	if err := s.waitOnDevicesAndCreateAliases(devs, "luks"); err != nil {
		return err
	}

	s.State.LuksPersistKeyFiles = make(map[string]string)
	s.State.LuksPersistSecureKeyRepoFiles = make(map[string]string)

	for _, luks := range config.Storage.Luks {
		// TODO: allow Ignition generated KeyFiles for
		// non-clevis devices that can be persisted.
		// TODO: create devices in parallel.
		// track whether Ignition creates the KeyFile
		// so that it can be removed
		if err := func() (err error) {
			var ignitionCreatedKeyFile bool
			devAlias := execUtil.DeviceAlias(*luks.Device)

			// create keyfile, remove on the way out
			keyFile, err := os.CreateTemp("", "ignition-luks-")
			if err != nil {
				return fmt.Errorf("creating keyfile: %w", err)
			}
			keyFilePath := keyFile.Name()
			if err := keyFile.Close(); err != nil {
				s.Warning("could not close file %s: %v", keyFilePath, err)
			}
			// We must pass 'keyFilePath' as a parameter here, since it may change below (cex).
			// Otherwise, the deferred function will see the modified value instead of the original one.
			defer func(name string) {
				if removeErr := os.Remove(name); removeErr != nil {
					s.Warning("could not remove file %s: %v", name, removeErr)
				}
			}(keyFilePath)

			if luks.Cex.IsPresent() {
				// each LUKS device has associated keyfiles
				keyFilePath = "/etc/luks/cex.key"
			} else if util.NilOrEmpty(luks.KeyFile.Source) {
				// generate keyfile contents
				key, err := randHex(4096)
				if err != nil {
					return fmt.Errorf("generating keyfile: %v", err)
				}
				if err := os.WriteFile(keyFilePath, []byte(key), 0400); err != nil {
					return fmt.Errorf("creating keyfile: %v", err)
				}
				ignitionCreatedKeyFile = true
			} else {
				f := types.File{
					Node: types.Node{
						Path: keyFilePath,
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: luks.KeyFile,
					},
				}
				fetchOps, err := s.PrepareFetches(s.Logger, f)
				if err != nil {
					return fmt.Errorf("failed to resolve keyfile %q: %v", f.Path, err)
				}
				for _, op := range fetchOps {
					if err := s.LogOp(
						func() error {
							return s.PerformFetch(op)
						}, "writing file %q", f.Path,
					); err != nil {
						return fmt.Errorf("failed to create keyfile %q: %v", op.Node.Path, err)
					}
				}
			}
			// store the key to be persisted into the real root
			// do this here so device reuse works correctly
			key, err := os.ReadFile(keyFilePath)
			if err != nil {
				return fmt.Errorf("failed to read keyfile %q: %w", keyFilePath, err)
			}
			s.State.LuksPersistKeyFiles[luks.Name] = dataurl.EncodeBytes(key)

			if !util.IsTrue(luks.WipeVolume) {
				// If the volume isn't forcefully being created, then we need
				// to check if it is of the correct type or that no volume exists.

				isLuks, err := s.isLuksDevice(*luks.Device)
				if err != nil {
					return err
				}
				if isLuks {
					// try to reuse the LUKS device; device will be opened
					// if successful.
					if err := s.reuseLuksDevice(luks, keyFilePath); err != nil {
						s.Err("volume wipe was not requested and luks device %q could not be reused: %v", *luks.Device, err)
						return ErrBadVolume
					}
					// Re-used devices cannot have Ignition generated key-files or be clevis devices so we cannot
					// leak any key files when exiting the loop early
					s.Info("volume at %q is already correctly formatted. Skipping...", *luks.Device)
					return nil
				}

				var info execUtil.FilesystemInfo
				err = s.LogOp(
					func() error {
						var err error
						info, err = execUtil.GetFilesystemInfo(devAlias, false)
						if err != nil {
							// Try again, allowing multiple filesystem
							// fingerprints this time.  If successful,
							// log a warning and continue.
							var err2 error
							info, err2 = execUtil.GetFilesystemInfo(devAlias, true)
							if err2 == nil {
								s.Warning("%v", err)
							}
							err = err2
						}
						return err
					},
					"determining volume type of %q", *luks.Device,
				)
				if err != nil {
					return err
				}
				s.Info("found %s at %q with uuid %q and label %q", info.Type, *luks.Device, info.UUID, info.Label)
				if info.Type != "" {
					s.Err("volume at %q is not of the correct type (found %s) and a volume wipe was not requested", *luks.Device, info.Type)
					return ErrBadVolume
				}
			} else {
				if _, err := s.LogCmd(
					exec.Command(distro.WipefsCmd(), "-a", devAlias),
					"wiping filesystem signatures from %q",
					devAlias,
				); err != nil {
					return fmt.Errorf("wipefs failed: %v", err)
				}
			}

			args := []string{
				"luksFormat",
				"--type", "luks2",
				"--key-file", keyFilePath,
			}

			if !util.NilOrEmpty(luks.Label) {
				args = append(args, "--label", *luks.Label)
			}

			if !util.NilOrEmpty(luks.UUID) {
				args = append(args, "--uuid", *luks.UUID)
			}

			if len(luks.Options) > 0 {
				// golang's a really great language...
				for _, option := range luks.Options {
					args = append(args, string(option))
				}
			}

			args = append(args, devAlias)

			// append the zkey specific luksFormat arguments
			if luks.Cex.IsPresent() {
				err := s.zkeySecKeyGen(luks)
				if err != nil {
					return fmt.Errorf("generating secure key: %w", err)
				}
				// Append the zkey cryptsetup specific parameter for luksFormat
				cex_args, err := s.zkeySecCryptGenArgs(luks)
				if err != nil {
					return fmt.Errorf("generating luksFormat args: %w", err)
				}
				args = append(args, cex_args...)
			}

			if _, err := s.LogCmd(
				exec.Command(distro.CryptsetupCmd(), args...),
				"creating %q", luks.Name,
			); err != nil {
				return fmt.Errorf("cryptsetup failed: %v", err)
			}

			if luks.Cex.IsPresent() {
				err := s.zkeyCryptSetvp(luks, keyFilePath)
				if err != nil {
					return err
				}
				if err := s.zkeySaveVolKeys(luks); err != nil {
					return fmt.Errorf("saving volume keys: %w ", err)
				}
			}

			// open the device
			if _, err := s.LogCmd(
				exec.Command(distro.CryptsetupCmd(), luksOpenArgs(luks, keyFilePath)...),
				"opening luks device %v", luks.Name,
			); err != nil {
				return fmt.Errorf("opening luks device: %v", err)
			}

			if luks.Clevis.IsPresent() {
				var pin string
				var config string

				if util.NotEmpty(luks.Clevis.Custom.Pin) {
					pin = *luks.Clevis.Custom.Pin
					config = *luks.Clevis.Custom.Config
				} else {
					// if the override pin is empty the config must also be empty
					pin = "sss"
					c := Clevis{
						Threshold: 1,
					}
					if luks.Clevis.Threshold != nil {
						c.Threshold = *luks.Clevis.Threshold
					}
					for _, tang := range luks.Clevis.Tang {
						var adv any
						if tang.Advertisement != nil {
							err := json.Unmarshal([]byte(*tang.Advertisement), &adv)
							if err != nil {
								return fmt.Errorf("unmarshalling advertisement: %v", err)
							}
						}
						c.Pins.Tang = append(c.Pins.Tang, Tang{
							URL:           tang.URL,
							Thumbprint:    *tang.Thumbprint,
							Advertisement: adv,
						})
					}
					if luks.Clevis.Tpm2 != nil {
						c.Pins.Tpm = *luks.Clevis.Tpm2
					}
					clevisJson, err := json.Marshal(c)
					if err != nil {
						return fmt.Errorf("creating clevis json: %v", err)
					}
					config = string(clevisJson)
				}

				// We cannot guarantee that networking is up yet, loop
				// through each tang device and fetch the server
				// advertisement to utilize Ignition's retry logic before we
				// pass the device to clevis. We have to loop each device as
				// the devices could be on different NICs that haven't come
				// up yet.

				// A running count of tang servers without an advertisement
				tangServersWithoutAdv := 0
				for _, tang := range luks.Clevis.Tang {
					u, err := url.Parse(tang.URL)
					if err != nil {
						return fmt.Errorf("parsing tang URL: %v", err)
					}
					if util.NilOrEmpty(tang.Advertisement) {
						tangServersWithoutAdv++
						u.Path = path.Join(u.Path, "adv")
						_, err = s.Fetcher.FetchToBuffer(*u, resource.FetchOptions{})
						if err != nil {
							return fmt.Errorf("fetching tang advertisement: %v", err)
						}
					}
				}

				// lets bind our device
				if _, err := s.LogCmd(
					exec.Command(distro.ClevisCmd(), "luks", "bind", "-f", "-k", keyFilePath, "-d", devAlias, pin, config), "Clevis bind",
				); err != nil {
					return fmt.Errorf("binding clevis device: %v", err)
				}
				intTpm2 := 0
				if util.IsTrue(luks.Clevis.Tpm2) {
					intTpm2 = 1
				}
				threshold := 1
				if luks.Clevis.Threshold != nil {
					threshold = *luks.Clevis.Threshold
				}
				// Check if we can safely close and re-open the device
				if tangServersWithoutAdv+intTpm2 >= threshold {
					// close & re-open Clevis devices to make sure that we can unlock them
					if _, err := s.LogCmd(
						exec.Command(distro.CryptsetupCmd(), "luksClose", luks.Name),
						"closing clevis luks device %v", luks.Name,
					); err != nil {
						return fmt.Errorf("closing luks device: %v", err)
					}
					if _, err := s.LogCmd(
						exec.Command(distro.ClevisCmd(), "luks", "unlock", "-d", devAlias, "-n", luks.Name),
						"reopening clevis luks device %s", luks.Name,
					); err != nil {
						return fmt.Errorf("reopening luks device %s: %v", luks.Name, err)
					}
				}
			}

			if ignitionCreatedKeyFile && luks.Clevis.IsPresent() {
				// assume the user does not want the generated key & remove it
				if _, err := s.LogCmd(
					exec.Command(distro.CryptsetupCmd(), "luksRemoveKey", devAlias, keyFilePath),
					"removing key file for %v", luks.Name,
				); err != nil {
					return fmt.Errorf("removing key file from luks device: %v", err)
				}
				delete(s.State.LuksPersistKeyFiles, luks.Name)
			}

			// It's best to wait here for the /dev/disk/by-*/X entries to be
			// (re)created, not only for other parts of the initramfs but
			// also because s.waitOnDevices() can still race with udev's
			// disk entry recreation.
			if err := s.waitForUdev(devAlias); err != nil {
				return fmt.Errorf("failed to wait for udev on %q after LUKS: %v", devAlias, err)
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
}

func (s *stage) isLuksDevice(device string) (bool, error) {
	checkLuks := func(luks2 bool) (bool, error) {
		ret := true
		desc := "luks"
		if luks2 {
			desc = "luks2"
		}
		err := s.LogOp(func() error {
			devAlias := execUtil.DeviceAlias(device)
			cmd := exec.Command(distro.CryptsetupCmd(), "isLuks", devAlias)
			if luks2 {
				cmd.Args = append(cmd.Args, "--type", "luks2")
			}
			cmdLine := log.QuotedCmd(cmd)
			s.Debug("executing: %s", cmdLine)

			if err := cmd.Run(); err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					ret = false
				} else {
					return fmt.Errorf("%w: Cmd: %s", err, cmdLine)
				}
			}
			return nil
		}, "checking if %v is a %v device", device, desc)
		return ret, err
	}

	// check for luks2
	isLuks, err := checkLuks(true)
	if isLuks || err != nil {
		return isLuks, err
	}
	// not luks2; check for luks1
	isLuks, err = checkLuks(false)
	if isLuks && err == nil {
		// we can't reuse the volume but it is LUKS.
		// fail to avoid data loss.
		return false, fmt.Errorf("%v is a luks device but not luks2; Ignition cannot reuse it", device)
	}
	return false, err
}

// Check LUKS device against config and open it.
func (s *stage) reuseLuksDevice(luks types.Luks, keyFilePath string) error {
	devAlias := execUtil.DeviceAlias(*luks.Device)

	// don't allow clevis device re-use as we can't guarantee that what
	// is stored in the header is exactly what would be generated from
	// the given config
	if luks.Clevis.IsPresent() {
		return fmt.Errorf("config must not specify clevis binding")
	}

	// ephemeral keyfiles won't match the existing device
	if util.NilOrEmpty(luks.KeyFile.Source) {
		return fmt.Errorf("config must specify keyfile")
	}

	if luks.Label != nil {
		fsInfo, err := execUtil.GetFilesystemInfo(devAlias, true)
		if err != nil {
			return fmt.Errorf("retrieving filesystem info: %v", err)
		}
		if fsInfo.Label != *luks.Label {
			return fmt.Errorf("volume label %q doesn't match expected label %q", fsInfo.Label, *luks.Label)
		}
	}

	if luks.UUID != nil {
		uuid, err := exec.Command(distro.CryptsetupCmd(), "luksUUID", devAlias).CombinedOutput()
		if err != nil {
			return err
		}
		uuidStr := strings.TrimSpace(string(uuid))
		if uuidStr != *luks.UUID {
			return fmt.Errorf("volume UUID %q doesn't match expected UUID %q", uuidStr, *luks.UUID)
		}
	}

	dump, err := newLuksDump(devAlias)
	if err != nil {
		return err
	}
	if dump.hasFlag("allow-discards") != util.IsTrue(luks.Discard) {
		return fmt.Errorf("volume allow-discards flag %v doesn't match expected value %v", dump.hasFlag("allow-discards"), util.IsTrue(luks.Discard))
	}

	// open the device to make sure the keyfile is valid
	if _, err := s.LogCmd(
		exec.Command(distro.CryptsetupCmd(), luksOpenArgs(luks, keyFilePath)...),
		"opening luks device %v", luks.Name,
	); err != nil {
		return fmt.Errorf("failed to open device using specified keyfile")
	}
	return nil
}

func luksOpenArgs(luks types.Luks, keyFilePath string) []string {
	ret := []string{
		"luksOpen",
		execUtil.DeviceAlias(*luks.Device),
		luks.Name,
		"--key-file",
		keyFilePath,
		// store presence/absence of --allow-discards and open options
		"--persistent",
	}
	if util.IsTrue(luks.Discard) {
		// clevis luks unlock doesn't have an option to enable
		// discard, so we persist the setting to the LUKS superblock
		// with --persistent, then omit it from the crypttab (since
		// crypttab would be misleading about where the setting is
		// really coming from).
		// https://github.com/latchset/clevis/issues/286
		ret = append(ret, "--allow-discards")
	}
	for _, opt := range luks.OpenOptions {
		// support persisting other options too
		ret = append(ret, string(opt))
	}
	return ret
}

type LuksDump struct {
	Config struct {
		Flags []string `json:"flags"`
	} `json:"config"`
}

func newLuksDump(devAlias string) (LuksDump, error) {
	dump, err := exec.Command(distro.CryptsetupCmd(), "luksDump", "--dump-json-metadata", devAlias).CombinedOutput()
	if err != nil {
		return LuksDump{}, err
	}
	var ret LuksDump
	if err := json.Unmarshal(dump, &ret); err != nil {
		return LuksDump{}, fmt.Errorf("parsing luks metadata: %w", err)
	}
	return ret, nil
}

func (d LuksDump) hasFlag(flag string) bool {
	for _, v := range d.Config.Flags {
		if v == flag {
			return true
		}
	}
	return false
}

// collect the cex card domains xx.xxxx
func getAllCexOnlineAPQNs() (string, error) {
	apDir := "/sys/bus/ap/devices/"
	var ret []string
	// cards details are in form of link file points to card directory.
	event, _ := os.ReadDir(apDir)
	if len(event) == 0 {
		return "", fmt.Errorf("adjunct processor not found")
	}
	// match the regexp for pattern xx.xxxx
	for _, dir := range event {
		if dir.IsDir() {
			continue
		}
		// if a match is found, read the uevent from that APQN directory
		// to verify whether it corresponds to a CCA controller. This is
		// intended for potential future use with an EP11 controller.
		if !cexRegx.MatchString(dir.Name()) {
			continue
		}
		filePath := filepath.Join(apDir, dir.Name(), "uevent")
		ctrl, err := getCryptoController(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse uevent file: %w", err)
		}
		for key := range ctrl {
			if ctrl[key] == "cca" {
				ret = append(ret, dir.Name())
			}
		}
	}
	if len(ret) == 0 {
		return "", fmt.Errorf("cannot find the cca ap controller")
	}
	return strings.Join(ret, ","), nil
}

// Find the AP (Adjunct Processor) mode from uevent.
// It may have CCA or EP11 CEX Controllers.
func getCryptoController(filePath string) (map[string]string, error) {
	apPairs := make(map[string]string)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading uevent file %q: %w", filePath, err)
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		apParts := strings.SplitN(line, "=", 2)
		if len(apParts) == 2 {
			key := strings.TrimSpace(apParts[0])
			value := strings.TrimSpace(apParts[1])
			apPairs[key] = value
		}
	}
	return apPairs, nil
}

// Arguments specific to zkey generation
func (s *stage) zkeySecGenArgs(luks types.Luks) []string {
	ret := []string{
		"generate",
		"--name",
		"ignition-luks-" + luks.Name,
		"--key-type",
		"CCA-AESCIPHER",
		"--xts",
		"--description",
		"Secure Key for " + luks.Name + " Volume",
	}
	return ret
}

// The secure key generated in the repository.
func (s *stage) zkeySecKeyGen(luks types.Luks) error {
	err := os.MkdirAll(distro.LuksRealVolumeKeyFilePath(), 0700)
	if err != nil {
		return fmt.Errorf("creating %q: %w", distro.LuksRealVolumeKeyFilePath(), err)
	}
	args := s.zkeySecGenArgs(luks)
	dom, err := getAllCexOnlineAPQNs()
	if err != nil {
		return fmt.Errorf("querying APQNs: %w", err)
	}
	args = append(args, "--apqns", dom)
	if _, err = s.LogCmd(
		exec.Command(distro.ZkeyCmd(), args...),
		"generating cex secure keys"); err != nil {
		return fmt.Errorf("generating secure key: %w", err)
	}
	return nil
}

// collect the Secure Key size
func (s *stage) zkeySecKeySize(f string) (string, error) {
	fileinfo, err := os.Stat(f)
	if err != nil {
		return "", fmt.Errorf("accessing file info %w", err)
	}
	// cryptsetup requires key size in bits.
	// to convert bytes to bits multiply the file size with 8.
	filesize := strconv.FormatInt(fileinfo.Size()*8, 10)

	return filesize, nil
}

// zkey Cryptsetup args generate pbkdf algo as argon2i.
// fips does not support argon2i. hence adding the
// args manually by verifying the keysize and keytype.
func (s *stage) zkeySecCryptGenArgs(luks types.Luks) ([]string, error) {
	zkeyfile := execUtil.ZkeySecureKeyRepoFiles(luks.Name)
	volKeyfile := path.Join(distro.LuksRealVolumeKeyFilePath(), zkeyfile[".skey"])
	ret := []string{
		"--volume-key-file",
		volKeyfile,
		"--cipher",
		"paes-xts-plain64",
	}
	keysize, err := s.zkeySecKeySize(volKeyfile)
	if err != nil {
		return ret, fmt.Errorf("error getting the key size %w", err)
	}
	ret = append(ret, "--key-size", keysize)

	return ret, nil
}

// Setting Verification pattern for keyslot after the luksFormat the device.
func (s *stage) zkeyCryptSetvp(luks types.Luks, key string) error {
	if _, err := s.LogCmd(exec.Command(distro.ZkeyCryptCmd(), "setvp",
		execUtil.DeviceAlias(*luks.Device),
		"--key-file", key),
		"Setting verification pattern for device: %q", execUtil.DeviceAlias(*luks.Device)); err != nil {
		return fmt.Errorf("zkey Verification pattern failed: %w", err)
	}
	return nil
}

// Save the LUKS volume key and info file into the root
func (s *stage) zkeySaveVolKeys(luks types.Luks) error {
	zkeyfile := execUtil.ZkeySecureKeyRepoFiles(luks.Name)
	zfilePath := distro.LuksRealVolumeKeyFilePath()
	for _, zfile := range zkeyfile {
		key, err := os.ReadFile(path.Join(zfilePath + zfile))
		if err != nil {
			return fmt.Errorf("failed to read volume key file %q: %w", zfile, err)
		}
		s.State.LuksPersistSecureKeyRepoFiles[zfile] = dataurl.EncodeBytes(key)
	}
	return nil
}
