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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	ut "github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/vcontext/report"
)

const (
	configDriveUserdataPath     = "/openstack/latest/user_data"
	metadataServiceMetadataPath = "openstack/latest/meta_data.json"
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
	urls := []url.URL{userdataURLs[resource.IPv4]}

	ipv6Interfaces, err := findInterfacesWithIPv6()
	if err != nil {
		// Fall back to IPv4 only
		f.Logger.Info("No active IPv6 network interface found: %v", err)
	}

	for _, ifaceName := range ipv6Interfaces {
		ipv6Url := userdataURLs[resource.IPv6]
		ipv6Url.Host = strings.Replace(ipv6Url.Host, "iface", ifaceName, 1)
		urls = append(urls, ipv6Url)
	}

	return fetchConfigParallel(f, urls)
}

// fetchUserdata fetches the userdata at the given URL. It returns no data
// if the metadata service doesn't have any userdata for the instance.
func fetchUserdata(f *resource.Fetcher, userdataURL url.URL) ([]byte, error) {
	data, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{})
	if !errors.Is(err, resource.ErrNotFound) {
		return data, err
	}

	// The metadata service returns 404 for every path until it knows about
	// the instance, and 404 for the userdata if none was provided. Wait for
	// the instance metadata, which always exists, to tell the two apart.
	metadataURL := userdataURL
	metadataURL.Path = metadataServiceMetadataPath
	_, err = f.FetchToBuffer(metadataURL, resource.FetchOptions{
		RetryCodes: []int{http.StatusNotFound},
	})
	if err != nil {
		return nil, err
	}

	data, err = f.FetchToBuffer(userdataURL, resource.FetchOptions{})
	if err != nil {
		// the metadata service knows about the instance but doesn't have
		// any userdata, assume that there is no config specified
		if errors.Is(err, resource.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return data, nil
}

// fetchConfigParallel fetches the userdata from all the URLs in parallel and
// returns the first successful result. It only fails once every URL has
// failed.
func fetchConfigParallel(f *resource.Fetcher, urls []url.URL) ([]byte, error) {
	type result struct {
		url  url.URL
		data []byte
		err  error
	}

	results := make(chan result, len(urls))
	for _, u := range urls {
		go func() {
			data, err := fetchUserdata(f, u)
			results <- result{url: u, data: data, err: err}
		}()
	}

	var err error
	for range urls {
		res := <-results
		if res.err == nil {
			f.Logger.Debug("got configuration from: %s", res.url.String())
			return res.data, nil
		}
		f.Logger.Err("fetching configuration for %s: %v", res.url.String(), res.err)
		err = res.err
	}

	f.Logger.Debug("all routines have failed to fetch configuration, returning last known error: %v", err)
	return nil, err
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
