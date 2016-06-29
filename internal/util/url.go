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

package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"syscall"

	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/systemd"

	"github.com/vincent-petithory/dataurl"
)

var (
	ErrSchemeUnsupported = errors.New("unsupported source scheme")
	ErrPathNotAbsolute   = errors.New("path is not absolute")
	ErrNotFound          = errors.New("resource not found")
	ErrFailed            = errors.New("failed to fetch resource")
)

const (
	oemDevicePath = "/dev/disk/by-label/OEM" // Device link where oem partition is found.
	oemDirPath    = "/usr/share/oem"         // OEM dir within root fs to consider for pxe scenarios.
	oemMountPath  = "/mnt/oem"               // Mountpoint where oem partition is mounted when present.
)

// FetchResource fetches a resource given a URL. The supported schemes are http(s), data, and oem.
// client is only needed if the scheme is http(s)
func FetchResource(l *log.Logger, client HttpClient, u url.URL) ([]byte, error) {
	var data []byte

	dataReader, err := FetchResourceAsReader(l, client, u)

	if err != nil {
		return nil, err
	}

	defer dataReader.Close()

	data, err = ioutil.ReadAll(dataReader)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Calls umount() after close
type ReadUnmounter struct {
	io.ReadCloser
	logger *log.Logger
}

func (f ReadUnmounter) Close() error {
	defer umountOEM(f.logger)
	return f.ReadCloser.Close()
}

// Returns a reader to the data at the url specified. Caller is responsible for
// closing the reader. client is only necessary if the url is http or https
func FetchResourceAsReader(l *log.Logger, client HttpClient, u url.URL) (io.ReadCloser, error) {
	switch u.Scheme {
	case "http", "https":
		dataReader, status, err := client.GetReader(u.String())
		if err != nil {
			return nil, err
		}

		l.Debug("GET result: %s", http.StatusText(status))
		switch status {
		case http.StatusOK, http.StatusNoContent:
			return dataReader, nil
		case http.StatusNotFound:
			return nil, ErrNotFound
		default:
			return nil, ErrFailed
		}

	case "data":
		url, err := dataurl.DecodeString(u.String())
		if err != nil {
			return nil, err
		}
		return ioutil.NopCloser(bytes.NewReader(url.Data)), nil

	case "oem":
		path := filepath.Clean(u.Path)
		if !filepath.IsAbs(path) {
			l.Err("oem path is not absolute: %q", u.Path)
			return nil, ErrPathNotAbsolute
		}

		// check if present under oemDirPath, if so use it.
		absPath := filepath.Join(oemDirPath, path)
		f, err := os.Open(absPath)
		if err == nil {
			return f, nil
		}
		if !os.IsNotExist(err) {
			l.Err("failed to read oem config: %v", err)
			return nil, ErrFailed
		}

		l.Info("oem config not found in %q, trying %q",
			oemDirPath, oemMountPath)

		// try oemMountPath, requires mounting it.
		err = mountOEM(l)
		if err != nil {
			l.Err("failed to mount oem partition: %v", err)
			return nil, ErrFailed
		}

		absPath = filepath.Join(oemMountPath, path)
		f, err = os.Open(absPath)
		if err != nil {
			l.Err("failed to read oem config: %v", err)
			umountOEM(l)
			return nil, ErrFailed
		}

		return ReadUnmounter{
			logger:     l,
			ReadCloser: f,
		}, nil

	case "":
		f, err := os.Open(os.DevNull)
		if err != nil {
			l.Err("Failed to open /dev/null for writing empty files")
			return nil, ErrFailed
		}
		return f, nil

	default:
		return nil, ErrSchemeUnsupported
	}
}

// mountOEM waits for the presence of and mounts the oem partition at oemMountPath.
func mountOEM(l *log.Logger) error {
	dev := []string{oemDevicePath}
	if err := systemd.WaitOnDevices(dev, "oem-cmdline"); err != nil {
		l.Err("failed to wait for oem device: %v", err)
		return err
	}

	if err := os.MkdirAll(oemMountPath, 0700); err != nil {
		l.Err("failed to create oem mount point: %v", err)
		return err
	}

	if err := l.LogOp(
		func() error {
			return syscall.Mount(dev[0], oemMountPath, "ext4", 0, "")
		},
		"mounting %q at %q", oemDevicePath, oemMountPath,
	); err != nil {
		return fmt.Errorf("failed to mount device %q at %q: %v",
			oemDevicePath, oemMountPath, err)
	}

	return nil
}

// umountOEM unmounts the oem partition at oemMountPath.
func umountOEM(l *log.Logger) {
	l.LogOp(
		func() error { return syscall.Unmount(oemMountPath, 0) },
		"unmounting %q", oemMountPath,
	)
}
