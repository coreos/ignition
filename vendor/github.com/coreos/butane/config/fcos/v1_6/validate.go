// Copyright 2020 Red Hat, Inc
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
// limitations under the License.)

package v1_6

import (
	"regexp"
	"strings"

	"github.com/coreos/butane/config/common"
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

const rootDevice = "/dev/disk/by-id/coreos-boot-disk"

var allowedMountpoints = regexp.MustCompile(`^/(etc|var)(/|$)`)
var dasdRe = regexp.MustCompile("(/dev/dasd[a-z]$)")
var sdRe = regexp.MustCompile("(/dev/sd[a-z]$)")

// We can't define a Validate function directly on Disk because that's defined in base,
// so we use a Validate function on the top-level Config instead.
func (conf Config) Validate(c path.ContextPath) (r report.Report) {
	for i, disk := range conf.Storage.Disks {
		if disk.Device != rootDevice && !util.IsTrue(disk.WipeTable) {
			for p, partition := range disk.Partitions {
				if partition.Number == 0 && partition.Label != nil {
					r.AddOnWarn(c.Append("storage", "disks", i, "partitions", p, "number"), common.ErrReuseByLabel)
				}
			}
		}
	}
	for i, fs := range conf.Storage.Filesystems {
		if fs.Path != nil && !allowedMountpoints.MatchString(*fs.Path) && util.IsTrue(fs.WithMountUnit) {
			r.AddOnError(c.Append("storage", "filesystems", i, "path"), common.ErrMountPointForbidden)
		}
	}
	return
}

func (d BootDevice) Validate(c path.ContextPath) (r report.Report) {
	if d.Layout != nil {
		switch *d.Layout {
		case "aarch64", "ppc64le", "x86_64":
			// Nothing to do
		case "s390x-eckd":
			if util.NilOrEmpty(d.Luks.Device) {
				r.AddOnError(c.Append("layout"), common.ErrNoLuksBootDevice)
			} else if !dasdRe.MatchString(*d.Luks.Device) {
				r.AddOnError(c.Append("layout"), common.ErrLuksBootDeviceBadName)
			}
		case "s390x-zfcp":
			if util.NilOrEmpty(d.Luks.Device) {
				r.AddOnError(c.Append("layout"), common.ErrNoLuksBootDevice)
			} else if !sdRe.MatchString(*d.Luks.Device) {
				r.AddOnError(c.Append("layout"), common.ErrLuksBootDeviceBadName)
			}
		case "s390x-virt":
		default:
			r.AddOnError(c.Append("layout"), common.ErrUnknownBootDeviceLayout)
		}

		// Mirroring the boot disk is not supported on s390x
		if strings.HasPrefix(*d.Layout, "s390x") && len(d.Mirror.Devices) > 0 {
			r.AddOnError(c.Append("layout"), common.ErrMirrorNotSupport)
		}
	}

	// CEX is only valid on s390x and incompatible with Clevis
	if util.IsTrue(d.Luks.Cex.Enabled) {
		if d.Layout == nil {
			r.AddOnError(c.Append("luks", "cex"), common.ErrCexArchitectureMismatch)
		} else if !strings.HasPrefix(*d.Layout, "s390x") {
			r.AddOnError(c.Append("layout"), common.ErrCexArchitectureMismatch)
		}
		if len(d.Luks.Tang) > 0 || util.IsTrue(d.Luks.Tpm2) {
			r.AddOnError(c.Append("luks"), errors.ErrCexWithClevis)
		}
	}

	r.Merge(d.Mirror.Validate(c.Append("mirror")))
	return
}

func (m BootDeviceMirror) Validate(c path.ContextPath) (r report.Report) {
	if len(m.Devices) == 1 {
		r.AddOnError(c.Append("devices"), common.ErrTooFewMirrorDevices)
	}
	return
}

func (user GrubUser) Validate(c path.ContextPath) (r report.Report) {
	if user.Name == "" {
		r.AddOnError(c.Append("name"), common.ErrGrubUserNameNotSpecified)
	}

	if !util.NotEmpty(user.PasswordHash) {
		r.AddOnError(c.Append("password_hash"), common.ErrGrubPasswordNotSpecified)
	}
	return
}
