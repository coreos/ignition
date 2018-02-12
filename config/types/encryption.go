// Copyright 2018 CoreOS, Inc.
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
	"errors"
	"fmt"

	"github.com/coreos/ignition/config/validate/report"
)

const (
	// encDevicesMax is the maximum number of encrypted volumes that can be
	// created at once. It is an arbitrary sane number for input validation,
	// it can be bumped if required.
	encDevicesMax = 1024
	// encKeyslotsMax is the maximux number of keyslots allowed per encrypted volume
	encKeyslotsMax = 1 // TODO(lucab): this could be expanded to 8
)

var (
	// ErrNoKeyslots is reported when 0 keyslots are specified
	ErrNoKeyslots = errors.New("no keyslots specified")
	// ErrNoKeyslotConfig is reported when a keyslot has no configured source
	ErrNoKeyslotConfig = errors.New("keyslot is missing provider configuration")
	// ErrTooManyKeyslotConfigs is reported when a keyslot has too many configured sources
	ErrTooManyKeyslotConfigs = errors.New("keyslot has multiple provider configurations")
	// ErrNoDevmapperName is reported when no device-mapper name is specified
	ErrNoDevmapperName = errors.New("missing device-mapper name")
	// ErrNoDevicePath is reported when no device path is specified
	ErrNoDevicePath = errors.New("missing device path")
	// ErrTooManyDevices is reported when too many devices are specified
	ErrTooManyDevices = fmt.Errorf("too many devices specified, at most %d allowed", encDevicesMax)
	// ErrTooManyKeyslots is reported when too many keyslots are specified
	ErrTooManyKeyslots = fmt.Errorf("too many keyslots specified, at most %d allowed", encKeyslotsMax)
)

// ValidateKeySlots validates the "keySlots" field
func (e Encryption) ValidateKeySlots() report.Report {
	r := report.Report{}
	if len(e.KeySlots) == 0 {
		r.Add(report.Entry{
			Message: ErrNoKeyslots.Error(),
			Kind:    report.EntryError,
		})
	}
	if len(e.KeySlots) > encKeyslotsMax {
		r.Add(report.Entry{
			Message: ErrTooManyKeyslots.Error(),
			Kind:    report.EntryError,
		})
	}

	for _, ks := range e.KeySlots {
		ksConfigured := 0
		if ks.Content != nil {
			ksConfigured++
		}
		// TODO(lucab): validate new providers here.
		if ksConfigured == 0 {
			r.Add(report.Entry{
				Message: ErrNoKeyslotConfig.Error(),
				Kind:    report.EntryError,
			})
		} else if ksConfigured > 1 {
			r.Add(report.Entry{
				Message: ErrTooManyKeyslotConfigs.Error(),
				Kind:    report.EntryError,
			})
		}
	}

	return r
}

// ValidateName validates the "name" field
func (e Encryption) ValidateName() report.Report {
	r := report.Report{}
	if e.Name == "" {
		r.Add(report.Entry{
			Message: ErrNoDevmapperName.Error(),
			Kind:    report.EntryError,
		})
	}
	return r
}

// ValidateDevice validates the "device" field
func (e Encryption) ValidateDevice() report.Report {
	r := report.Report{}
	if e.Device == "" {
		r.Add(report.Entry{
			Message: ErrNoDevicePath.Error(),
			Kind:    report.EntryError,
		})
	}
	return r
}

// ValidateSource validates the "source" field
func (c Content) ValidateSource() report.Report {
	r := report.Report{}
	if err := validateURL(c.Source, "https"); err != nil {
		r.Add(report.Entry{
			Message: err.Error(),
			Kind:    report.EntryError,
		})
	}
	return r
}
