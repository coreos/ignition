// Copyright 2020 Red Hat, Inc.
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

package types

import (
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func (f Filesystem) Key() string {
	return f.Device
}

func (f Filesystem) IgnoreDuplicates() map[string]struct{} {
	return map[string]struct{}{
		"Options":      {},
		"MountOptions": {},
	}
}

func (f Filesystem) Validate(c path.ContextPath) (r report.Report) {
	r.AddOnError(c.Append("path"), f.validatePath())
	r.AddOnError(c.Append("device"), f.validateDevice())
	r.AddOnError(c.Append("format"), f.validateFormat())
	r.AddOnError(c.Append("label"), f.validateLabel())

	f.validateBlockDeviceOnlyFields(c, &r)

	return
}

// Non-block file systems cannot have UUID, Mkfs options, or wipeFilesystem enabled
func (f Filesystem) validateBlockDeviceOnlyFields(c path.ContextPath, r *report.Report) {
	if f.IsBlockDevice() {
		return
	}
	if util.NotEmpty(f.UUID) {
		r.AddOnError(c.Append("uuid"), errors.ErrUUIDNotSupportedForFormat)
	}
	if util.IsTrue(f.WipeFilesystem) {
		r.AddOnError(c.Append("wipeFilesystem"), errors.ErrWipeNotSupportedForFormat)
	}
	if len(f.Options) > 0 {
		r.AddOnError(c.Append("options"), errors.ErrMkfsOptionsNotSupportedForFormat)
	}
}

func (f Filesystem) IsBlockDevice() bool {
	if util.NilOrEmpty(f.Format) {
		return true
	}

	switch *f.Format {
	case "virtiofs":
		return false
	default:
		return true
	}
}

func (f Filesystem) validateDevice() error {
	if util.NilOrEmpty(f.Format) {
		return validatePath(f.Device)
	}

	switch *f.Format {
	// File Systems that use tags don't need a path check, lets just make sure it is not empty
	case "virtiofs":
		if f.Device == "" {
			return errors.ErrNoDevice
		}
		// This is part of the virtiofs spec. QEMU should also error if you try to startup with a tag too long
		if len(f.Device) > 36 {
			return errors.ErrVirtiofsDeviceTagTooLong
		}
		return nil
	default:
		return validatePath(f.Device)
	}
}

func (f Filesystem) validatePath() error {
	return validatePathNilOK(f.Path)
}

func (f Filesystem) validateFormat() error {
	if util.NilOrEmpty(f.Format) {
		if util.NotEmpty(f.Path) ||
			util.NotEmpty(f.Label) ||
			util.NotEmpty(f.UUID) ||
			util.IsTrue(f.WipeFilesystem) ||
			len(f.MountOptions) != 0 ||
			len(f.Options) != 0 {
			return errors.ErrFormatNilWithOthers
		}
	} else {
		switch *f.Format {
		case "ext4", "btrfs", "xfs", "swap", "vfat", "none", "virtiofs":
		default:
			return errors.ErrFilesystemInvalidFormat
		}
	}
	return nil
}

func (f Filesystem) validateLabel() error {
	if util.NilOrEmpty(f.Label) {
		return nil
	}
	if util.NilOrEmpty(f.Format) {
		return errors.ErrLabelNeedsFormat
	}

	switch *f.Format {
	case "ext4":
		if len(*f.Label) > 16 {
			// source: man mkfs.ext4
			return errors.ErrExt4LabelTooLong
		}
	case "btrfs":
		if len(*f.Label) > 256 {
			// source: man mkfs.btrfs
			return errors.ErrBtrfsLabelTooLong
		}
	case "xfs":
		if len(*f.Label) > 12 {
			// source: man mkfs.xfs
			return errors.ErrXfsLabelTooLong
		}
	case "swap":
		// mkswap's man page does not state a limit on label size, but through
		// experimentation it appears that mkswap will truncate long labels to
		// 15 characters, so let's enforce that.
		if len(*f.Label) > 15 {
			return errors.ErrSwapLabelTooLong
		}
	case "vfat":
		if len(*f.Label) > 11 {
			// source: man mkfs.fat
			return errors.ErrVfatLabelTooLong
		}
	case "virtiofs":
		// This will only be reached if the label is non-empty
		return errors.ErrVirtiofsCannotHaveLabel
	}
	return nil
}
