// Copyright 2016 CoreOS, Inc.
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

// The OpenStack provider fetches configurations from the userdata available in
// both the config-drive as well as the network metadata service. Whichever
// responds first is the config that is used.
// NOTE: This provider is still EXPERIMENTAL.

package openstack

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	ut "github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/vcontext/report"
)

const (
	configDriveUserdataPath = "/openstack/latest/user_data"
)

var (
	metadataServiceUrlIPv4 = url.URL{
		Scheme: "http",
		Host:   "169.254.169.254",
		Path:   "openstack/latest/user_data",
	}
	metadataServiceUrlIPv6 = url.URL{
		Scheme: "http",
		Host:   "[fe80::a9fe:a9fe%iface]",
		Path:   "openstack/latest/user_data",
	}
)

func init() {
	platform.Register(platform.Provider{
		Name:  "openstack",
		Fetch: fetchConfig,
	})
	// the brightbox platform ID just uses the OpenStack provider code
	platform.Register(platform.Provider{
		Name:  "brightbox",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
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

	dispatch("config drive (config-2)", func() ([]byte, error) {
		return fetchConfigFromDevice(f.Logger, ctx, filepath.Join(distro.DiskByLabelDir(), "config-2"))
	})

	dispatch("config drive (CONFIG-2)", func() ([]byte, error) {
		return fetchConfigFromDevice(f.Logger, ctx, filepath.Join(distro.DiskByLabelDir(), "CONFIG-2"))
	})

	dispatch("metadata service", func() ([]byte, error) {
		return fetchConfigFromMetadataService(f)
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

func fetchConfigFromDevice(logger *log.Logger, ctx context.Context, path string) ([]byte, error) {
	for !fileExists(path) {
		logger.Debug("config drive (%q) not found. Waiting...", path)
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	logger.Debug("creating temporary mount point")
	mnt, err := os.MkdirTemp("", "ignition-configdrive")
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

	return os.ReadFile(filepath.Join(mnt, configDriveUserdataPath))
}

// checks if the given IP is an IPv6 address.
func isIPv6Address(ip net.IP) bool {
	isIPv6 := ip.To4() == nil
	return isIPv6
}

// findNetworkInterfaceWithIPv6 returns the name of the first network interface with an active IPv6 address.
// This interface name is needed to format the link-local address for accessing the IPv6 metadata service.
// For more details, see: https://docs.openstack.org/nova/2024.1/user/metadata.html
func findNetworkInterfaceWithIPv6() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("error fetching network interfaces: %v", err)
	}

	for _, iface := range interfaces {

		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("Error fetching addresses for interface %s: %v\n", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && isIPv6Address(ipnet.IP) {
				// Prepare the IPv6 metadata service URL
				metadataServiceUrlIPv6.Host = strings.Replace(metadataServiceUrlIPv6.Host, "iface", iface.Name, 1)
				return nil
			}
		}
	}
	return fmt.Errorf("no active IPv6 network interface found")
}

// Fetches configuration from IPv4 and IPv6 metadata services
func fetchConfigFromMetadataService(f *resource.Fetcher) ([]byte, error) {
	var response []byte
	var err error

	// Try fetching from IPv4 first
	response, err = f.FetchToBuffer(metadataServiceUrlIPv4, resource.FetchOptions{})
	f.Logger.Debug("IPv6 URL:", metadataServiceUrlIPv6.Host)
	f.Logger.Debug("IPv4 URL:", metadataServiceUrlIPv4)

	if err != nil {
		f.Logger.Info("IPv4 fetch failed: %v. Attempting to fetch from IPv6...\n", err)

		// If IPv4 fails, find the network interface for IPv6
		err := findNetworkInterfaceWithIPv6()
		if err != nil {
			f.Logger.Info("IPv6 metadata service lookup failed: %v\n", err)
			return nil, fmt.Errorf("both IPv4 and IPv6 lookup failed")
		}
		f.Logger.Debug("Fetching from IPv6 metadata service at %s...\n", metadataServiceUrlIPv6.String())

		// Try to fetch from IPv6
		f.Logger.Debug("IPv6 URL:", metadataServiceUrlIPv6.Host)
		f.Logger.Debug("Attempting to fetch from IPv6 metadata service at: %s\n", metadataServiceUrlIPv6.String())

		response, err = f.FetchToBuffer(metadataServiceUrlIPv6, resource.FetchOptions{})
		if err != nil {
			return nil, fmt.Errorf("IPv4 and IPv6 services failed")
		}

		f.Logger.Debug("Successfully fetched configuration from IPv6.")
		return response, nil
	}

	return response, nil

}
