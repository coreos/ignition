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

//go:build !s390x && !ppc64le

// The default QEMU provider fetches a local configuration from the firmware
// config interface (opt/com.coreos/config). Platforms without support for
// qemu_fw_cfg should use the blockdev implementation instead.

package qemu

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	firmwareConfigPath     = "/sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config/raw"
	firmwareConfigSizePath = "/sys/firmware/qemu_fw_cfg/by_name/opt/com.coreos/config/size"
)

func init() {
	platform.Register(platform.Provider{
		Name:  "qemu",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (cfg types.Config, rpt report.Report, err error) {
	// load qemu_fw_cfg module
	if _, statErr := os.Stat("/sys/firmware/qemu_fw_cfg"); statErr != nil {
		if _, err = f.Logger.LogCmd(exec.Command(distro.ModprobeCmd(), "qemu_fw_cfg"), "loading QEMU firmware config module"); err != nil {
			return
		}
	}

	// get size of firmware blob, if it exists
	var sizeBytes []byte
	sizeBytes, err = os.ReadFile(firmwareConfigSizePath)
	if os.IsNotExist(err) {
		f.Logger.Info("QEMU firmware config was not found. Ignoring...")
		cfg, rpt, err = util.ParseConfig(f.Logger, []byte{})
		return
	} else if err != nil {
		f.Logger.Err("couldn't read QEMU firmware config size: %v", err)
		return
	}
	var size int
	size, err = strconv.Atoi(strings.TrimSpace(string(sizeBytes)))
	if err != nil {
		f.Logger.Err("couldn't parse QEMU firmware config size: %v", err)
		return
	}

	// Read firmware blob.  We need to make as few, large read() calls as
	// possible, since the qemu_fw_cfg kernel module takes O(offset)
	// time for each read syscall.  os.ReadFile() would eventually
	// converge on the correct read size (one page) but we can do
	// better, and without reallocating.
	// Leave an extra guard byte to check for EOF
	data := make([]byte, 0, size+1)
	var fh *os.File
	fh, err = os.Open(firmwareConfigPath)
	if err != nil {
		f.Logger.Err("couldn't open QEMU firmware config: %v", err)
		return
	}
	defer func() {
		_ = fh.Close()
	}()
	lastReport := time.Now()
	reporting := false
	for len(data) < size {
		// if size is correct, we will never call this at an offset
		// where it would return io.EOF
		var n int
		n, err = fh.Read(data[len(data):cap(data)])
		if err != nil {
			f.Logger.Err("couldn't read QEMU firmware config: %v", err)
			return
		}
		data = data[:len(data)+n]
		if !reporting && time.Since(lastReport).Seconds() >= 10 {
			f.Logger.Warning("Reading QEMU fw_cfg takes quadratic time. Consider moving large files or config fragments to a remote URL.")
			reporting = true
		}
		if reporting && (time.Since(lastReport).Seconds() >= 5 || len(data) >= size) {
			f.Logger.Info("Reading config from QEMU fw_cfg: %d/%d KB", len(data)/1024, size/1024)
			lastReport = time.Now()
		}
	}
	if len(data) > size {
		// overflowed into guard byte
		f.Logger.Err("missing EOF reading QEMU firmware config")
		err = errors.New("missing EOF")
		return
	}
	// If size is not at a page boundary, we know we're at EOF because
	// the guard byte was not filled.  If size is at a page boundary,
	// trust that firmwareConfigSizePath was telling the truth to avoid
	// incurring an extra read call to check for EOF.  We're at the end
	// of the file so the extra read would be maximally expensive.
	cfg, rpt, err = util.ParseConfig(f.Logger, data)
	return
}
