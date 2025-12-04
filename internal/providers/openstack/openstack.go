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
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
	userdataURLs = map[string]url.URL{
		resource.IPv4: {
			Scheme: "http",
			Host:   "169.254.169.254",
			Path:   "openstack/latest/user_data",
		},

		resource.IPv6: {
			Scheme: "http",
			Host:   "[fe80::a9fe:a9fe%iface]",
			Path:   "openstack/latest/user_data",
		},
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
	defer func() {
		if removeErr := os.Remove(mnt); removeErr != nil {
			logger.Warning("failed to remove temp directory %q: %v", mnt, removeErr)
		}
	}()

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

func fetchConfigFromMetadataService(f *resource.Fetcher) ([]byte, error) {
	ipv6Interfaces, err := findInterfacesWithIPv6()
	if err != nil {
		f.Logger.Info("No active IPv6 network interface found: %v", err)
		// Fall back to IPv4 only
		return fetchConfigFromMetadataServiceIPv4Only(f)
	}

	urls := []url.URL{userdataURLs[resource.IPv4]}

	for _, ifaceName := range ipv6Interfaces {
		ipv6Url := userdataURLs[resource.IPv6]
		ipv6Url.Host = strings.Replace(ipv6Url.Host, "iface", ifaceName, 1)
		urls = append(urls, ipv6Url)
	}

	// Use parallel fetching for all interfaces
	cfg, _, err := fetchConfigParallel(f, urls)

	// the metadata server exists but doesn't contain any actual metadata,
	// assume that there is no config specified
	if err == resource.ErrNotFound {
		return nil, nil
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func fetchConfigFromMetadataServiceIPv4Only(f *resource.Fetcher) ([]byte, error) {
	urls := map[string]url.URL{
		string(resource.IPv4): userdataURLs[resource.IPv4],
	}

	cfg, _, err := resource.FetchConfigDualStack(
		f,
		urls,
		func(f *resource.Fetcher, u url.URL) ([]byte, error) {
			return f.FetchToBuffer(u, resource.FetchOptions{})
		},
	)

	// the metadata server exists but doesn't contain any actual metadata,
	// assume that there is no config specified
	if err == resource.ErrNotFound {
		return nil, nil
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func fetchConfigParallel(f *resource.Fetcher, urls []url.URL) (types.Config, report.Report, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		err      error
		nbErrors int
	)

	cfg := make(map[url.URL][]byte)

	success := make(chan url.URL, 1)
	errors := make(chan error, len(urls))

	// Use waitgroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	fetch := func(_ context.Context, u url.URL) {
		defer wg.Done()
		d, e := f.FetchToBuffer(u, resource.FetchOptions{})
		if e != nil {
			f.Logger.Err("fetching configuration for %s: %v", u.String(), e)
			err = e
			errors <- e
		} else {
			cfg[u] = d
			select {
			case success <- u:
			default:
			}
		}
	}

	// Start goroutines for all URLs
	for _, u := range urls {
		wg.Add(1)
		go fetch(ctx, u)
	}

	// Wait for the first success or all failures
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case u := <-success:
		f.Logger.Debug("got configuration from: %s", u.String())
		return util.ParseConfig(f.Logger, cfg[u])
	case <-errors:
		nbErrors++
		if nbErrors == len(urls) {
			f.Logger.Debug("all routines have failed to fetch configuration, returning last known error: %v", err)
			return types.Config{}, report.Report{}, err
		}
	case <-done:
		// All goroutines completed, check if we have any success
		if len(cfg) > 0 {
			// Return the first successful configuration
			for u, data := range cfg {
				f.Logger.Debug("got configuration from: %s", u.String())
				return util.ParseConfig(f.Logger, data)
			}
		}
	}

	return types.Config{}, report.Report{}, err
}

func findInterfacesWithIPv6() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error fetching network interfaces: %v", err)
	}

	var ipv6Interfaces []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To16() != nil && ipnet.IP.To4() == nil {
				ipv6Interfaces = append(ipv6Interfaces, iface.Name)
				break
			}
		}
	}

	if len(ipv6Interfaces) == 0 {
		return nil, fmt.Errorf("no active IPv6 network interface found")
	}

	return ipv6Interfaces, nil
}
