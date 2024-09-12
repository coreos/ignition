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
	"time"

	"github.com/coreos/ignition/v2/config/v3_5_experimental/types"
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

// Checks if an IP is an IPv6 address
func isIPv6Address(ip net.IP) bool {
	isIPv6 := ip.To4() == nil
	fmt.Printf("Checking if IP is IPv6: %s, result: %v\n", ip.String(), isIPv6)
	return isIPv6
}

// Finds the first available IPv6 address on active interfaces
func findIPv6Address() (string, error) {
	fmt.Println("Fetching network interfaces...")
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("error fetching interfaces: %v", err)
	}

	for _, iface := range interfaces {
		fmt.Printf("Checking interface: %s\n", iface.Name)

		// Skip inactive or loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("error fetching addresses for interface %s: %v\n", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && isIPv6Address(ipnet.IP) {
				ipv6Address := fmt.Sprintf("[%s]", ipnet.IP.String())
				fmt.Printf("Found IPv6 address: %s on interface %s\n", ipv6Address, iface.Name)
				return ipv6Address, nil
			}
		}
	}
	return "", fmt.Errorf("no IPv6 address found")
}

// Fetches configuration from both IPv4 and IPv6 metadata services
func fetchConfigFromMetadataService(f *resource.Fetcher) ([]byte, error) {
	var ipv4Res, ipv6Res []byte
	var ipv4Err, ipv6Err error

	fmt.Println("Fetching from IPv4 metadata service...")
	ipv4Res, ipv4Err = f.FetchToBuffer(metadataServiceUrlIPv4, resource.FetchOptions{})
	if ipv4Err == nil {
		fmt.Println("Successfully fetched configuration from IPv4 metadata service.")

		// If IPv4 is successful, attempt to fetch IPv6
		fmt.Println("Fetching IPv6 address for metadata service...")
		ipv6Address, err := findIPv6Address()
		if err != nil {
			fmt.Printf("IPv6 metadata service lookup failed: %v\n", err)
			return ipv4Res, fmt.Errorf("IPv6 lookup failed, returning only IPv4 result")
		}

		metadataServiceUrlIPv6.Host = ipv6Address
		fmt.Printf("Fetching from IPv6 metadata service at %s...\n", metadataServiceUrlIPv6.String())
		ipv6Res, ipv6Err = f.FetchToBuffer(metadataServiceUrlIPv6, resource.FetchOptions{})

		if ipv6Err != nil {
			fmt.Printf("IPv6 metadata service failed: %v\n", ipv6Err)
			return ipv4Res, fmt.Errorf("IPv4 succeeded, IPv6 failed: %v", ipv6Err)
		}
		fmt.Println("Successfully fetched configuration from both IPv4 and IPv6.")
		return append(ipv4Res, ipv6Res...), nil
	}

	// If IPv4 fails, attempt to fetch IPv6
	fmt.Printf("IPv4 metadata service failed: %v\n", ipv4Err)
	fmt.Println("Trying to fetch from IPv6 metadata service...")

	ipv6Address, err := findIPv6Address()
	if err != nil {
		fmt.Printf("IPv6 metadata service lookup failed: %v\n", err)
		return nil, fmt.Errorf("both IPv4 and IPv6 lookup failed")
	}

	metadataServiceUrlIPv6.Host = ipv6Address
	fmt.Printf("Fetching from IPv6 metadata service at %s...\n", metadataServiceUrlIPv6.String())
	ipv6Res, ipv6Err = f.FetchToBuffer(metadataServiceUrlIPv6, resource.FetchOptions{})
	if ipv6Err != nil {
		fmt.Printf("IPv6 metadata service failed: %v\n", ipv6Err)
		return nil, fmt.Errorf("both IPv4 and IPv6 services failed")
	}

	fmt.Println("Successfully fetched configuration from IPv6 metadata service.")
	return ipv6Res, fmt.Errorf("IPv4 failed, returning IPv6 result")
}
