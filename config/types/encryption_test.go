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
	"reflect"
	"testing"

	"github.com/coreos/ignition/config/validate/report"
)

func TestEncryptionValidateName(t *testing.T) {
	simpleSlots := []Keyslot{
		{
			Content: &Content{Source: "https://localhost/key.txt"},
		},
	}

	tests := []struct {
		in  Encryption
		out error
	}{
		{
			in:  Encryption{Name: "foo", Device: "/dev/bar", KeySlots: simpleSlots},
			out: nil,
		},
		{
			in:  Encryption{Name: "", Device: "/dev/bar", KeySlots: simpleSlots},
			out: ErrNoDevmapperName,
		},
	}

	for i, tt := range tests {
		err := tt.in.ValidateName()
		if !reflect.DeepEqual(report.ReportFromError(tt.out, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, tt.out, err)
		}
	}
}

func TestEncryptionValidateDevice(t *testing.T) {
	simpleSlots := []Keyslot{
		{
			Content: &Content{Source: "https://localhost/key.txt"},
		},
	}

	tests := []struct {
		in  Encryption
		out error
	}{
		{
			in:  Encryption{Name: "foo", Device: "/dev/bar", KeySlots: simpleSlots},
			out: nil,
		},
		{
			in:  Encryption{Name: "foo", Device: "", KeySlots: simpleSlots},
			out: ErrNoDevicePath,
		},
	}

	for i, tt := range tests {
		err := tt.in.ValidateDevice()
		if !reflect.DeepEqual(report.ReportFromError(tt.out, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, tt.out, err)
		}
	}
}

func TestEncryptionValidateKeySlots(t *testing.T) {
	simpleSlots := []Keyslot{
		{
			Content: &Content{Source: "https://localhost/key.txt"},
		},
	}

	tests := []struct {
		in  Encryption
		out error
	}{
		{
			in:  Encryption{Name: "foo", Device: "/dev/bar", KeySlots: simpleSlots},
			out: nil,
		},
		{
			in:  Encryption{Name: "foo", Device: "/dev/bar", KeySlots: []Keyslot{}},
			out: ErrNoKeyslots,
		},
		{
			in:  Encryption{Name: "foo", Device: "/dev/bar", KeySlots: []Keyslot{{}}},
			out: ErrNoKeyslotConfig,
		},
	}

	for i, tt := range tests {
		err := tt.in.ValidateKeySlots()
		if !reflect.DeepEqual(report.ReportFromError(tt.out, report.EntryError), err) {
			t.Errorf("#%d: bad error: want %v, got %v", i, tt.out, err)
		}
	}
}
