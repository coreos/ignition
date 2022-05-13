// Copyright 2017 CoreOS, Inc.
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

// The CloudStack provider fetches configurations from the userdata available in
// both the config-drive as well as the network metadata service. Whichever
// responds first is the config that is used.
// NOTE: This provider is still EXPERIMENTAL.

package cloudstack

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/networkmanager"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	ut "github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/vcontext/report"
)

const (
	configDriveUserdataPath = "/cloudstack/userdata/user_data.txt"
	retryInterval           = 500 * time.Millisecond
	dataServerDNSName       = "data-server"
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	// The fetch-offline approach doesn't work well here because of the "split
	// personality" of this provider. See:
	// https://github.com/coreos/ignition/issues/1081
	if f.Offline {
		return types.Config{}, report.Report{}, resource.ErrNeedNet
	}

	var data []byte
	errChan := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	dispatchCount := 0

	dispatch := func(name string, fn func() ([]byte, error)) {
		dispatchCount++
		go func() {
			raw, err := fn()
			if err != nil {
				switch err {
				case context.Canceled:
				default:
					f.Logger.Err("failed to fetch config from %s: %v", name, err)
				}
				errChan <- err
				return
			}

			data = raw
			cancel()
		}()
	}

	dispatch("config drive (config)", func() ([]byte, error) {
		return fetchConfigFromDevice(f.Logger, ctx, "config-2")
	})

	dispatch("config drive (CONFIG)", func() ([]byte, error) {
		return fetchConfigFromDevice(f.Logger, ctx, "CONFIG-2")
	})

	dispatch("metadata service", func() ([]byte, error) {
		return fetchConfigFromMetadataService(ctx, f)
	})

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-errChan:
			dispatchCount--
			if dispatchCount == 0 {
				f.Logger.Info("couldn't fetch config")
				break Loop
			}
		}
	}

	return util.ParseConfig(f.Logger, data)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

func labelExists(label string) bool {
	_, err := getPath(label)
	return (err == nil)
}

func getPath(label string) (string, error) {
	path := filepath.Join(distro.DiskByLabelDir(), label)

	if fileExists(path) {
		return path, nil
	}

	return "", fmt.Errorf("label not found: %s", label)
}

func findLease() (*os.File, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("could not list interfaces: %v", err)
	}

	for _, iface := range ifaces {
		lease, err := os.Open(fmt.Sprintf("/run/systemd/netif/leases/%d", iface.Index))
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		} else {
			return lease, nil
		}
	}

	return nil, fmt.Errorf("no leases found")
}

func getVirtualRouterAddress(ctx context.Context, logger *log.Logger) (string, error) {
	for {
		// Try "data-server" DNS entry first
		if addr, err := getDataServerByDNS(); err != nil {
			logger.Info("Could not find virtual router using DNS: %s", err.Error())
			// continue with NetworkManager
		} else {
			logger.Info("Virtual router address found using DNS: %s", addr)
			return addr, nil
		}

		// Then use NetworkManager to get the server option via DHCP
		if addr, err := getNetworkManagerDHCPServerOption(); err != nil {
			logger.Info("Could not find virtual router using NetworkManager DHCP server option: %s", err.Error())
			// continue with networkd
		} else {
			logger.Info("Virtual router address found using NetworkManager DHCP server option: %s", addr)
			return addr, nil
		}

		// Then try networkd
		if addr, err := getDHCPServerAddress(); err != nil {
			logger.Info("Could not find server address in DHCP networkd leases: %s", err.Error())
			// continue with default gateway
		} else {
			logger.Info("Virtual router address found using DHCP networkd leases: %s", addr)
			return addr, nil
		}

		// Fallback on default gateway
		if addr, err := getDefaultGateway(); err != nil {
			logger.Info("Could not find default gateway: %s", err.Error())
		} else {
			logger.Info("Fallback on default gateway: %s", addr)
			return addr, nil
		}

		select {
		case <-time.After(retryInterval):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func getDataServerByDNS() (string, error) {
	addrs, err := net.LookupHost(dataServerDNSName)
	if err != nil {
		return "", fmt.Errorf("could not execute DNS lookup: %v", err)
	}

	for _, addr := range addrs {
		return addr, nil
	}
	return "", fmt.Errorf("DNS Entry %s not found", dataServerDNSName)
}

func getNetworkManagerDHCPServerOption() (string, error) {
	options, err := networkmanager.GetDHCPOptions()
	if err != nil {
		return "", err
	}
	for _, netIface := range options {
		for k, v := range netIface {
			if k == "dhcp_server_identifier" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("no DHCP option dhcp_server_identifier in NetworkManager")
}

func getDHCPServerAddress() (string, error) {
	lease, err := findLease()
	if err != nil {
		return "", err
	}
	defer lease.Close()

	var address string
	line := bufio.NewScanner(lease)
	for line.Scan() {
		parts := strings.Split(line.Text(), "=")
		if parts[0] == "SERVER_ADDRESS" && len(parts) == 2 {
			address = parts[1]
			break
		}
	}

	if len(address) == 0 {
		return "", fmt.Errorf("dhcp server address not found in leases")
	}

	return address, nil
}

func getDefaultGateway() (string, error) {
	file, err := os.Open(distro.RouteFilePath())
	if err != nil {
		return "", fmt.Errorf("cannot read routes: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore headers
		if strings.HasPrefix(line, "Iface") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			return "", fmt.Errorf("cannot parse route files")
		}
		if fields[1] == "00000000" {
			// destination is "0.0.0.0", so the gateway is the default gateway
			gw, err := parseIP(fields[2])
			if err != nil {
				return "", fmt.Errorf("cannot parse route files: %v", err)
			}
			return gw, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("cannot parse route files: %v", err)
	}

	return "", fmt.Errorf("default gateway not found")
}

// parseIP takes the reverse hex IP address string from rooute
// file and converts it to dotted decimal IPv4 format.
func parseIP(str string) (string, error) {
	if str == "" {
		return "", fmt.Errorf("input is empty")
	}
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return "", err
	}
	if len(bytes) != 4 {
		return "", fmt.Errorf("invalid IPv4 address %s", str)
	}
	return net.IPv4(bytes[3], bytes[2], bytes[1], bytes[0]).String(), nil
}

func fetchConfigFromDevice(logger *log.Logger, ctx context.Context, label string) ([]byte, error) {
	for !labelExists(label) {
		logger.Debug("config drive (%q) not found. Waiting...", label)
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	path, err := getPath(label)
	if err != nil {
		return nil, err
	}

	logger.Debug("creating temporary mount point")
	mnt, err := ioutil.TempDir("", "ignition-configdrive")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.Remove(mnt)

	cmd := exec.Command(distro.MountCmd(), "-o", "ro", "-t", "auto", path, mnt)
	if _, err := logger.LogCmd(cmd, "mounting config drive"); err != nil {
		return nil, err
	}
	defer func() {
		_ = logger.LogOp(
			func() error {
				return ut.UmountPath(mnt)
			},
			"unmounting %q at %q", path, mnt,
		)
	}()

	if !fileExists(filepath.Join(mnt, configDriveUserdataPath)) {
		return nil, nil
	}

	return ioutil.ReadFile(filepath.Join(mnt, configDriveUserdataPath))
}

func fetchConfigFromMetadataService(ctx context.Context, f *resource.Fetcher) ([]byte, error) {
	addr, err := getVirtualRouterAddress(ctx, f.Logger)
	if err != nil {
		return nil, err
	}

	metadataServiceURL := url.URL{
		Scheme: "http",
		Host:   addr,
		Path:   "/latest/user-data",
	}

	res, err := f.FetchToBuffer(metadataServiceURL, resource.FetchOptions{})

	// the metadata server exists but doesn't contain any actual metadata,
	// assume that there is no config specified
	if err == resource.ErrNotFound {
		return nil, nil
	}

	return res, err
}
