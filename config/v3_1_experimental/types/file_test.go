// Copyright 2019 Red Hat, Inc.
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

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/validate/report"
)

func TestFileValidateOverwrite(t *testing.T) {
	tests := []struct {
		in  File
		out error
	}{
		{
			in:  File{},
			out: nil,
		},
		{
			in: File{
				Node: Node{
					Overwrite: util.BoolToPtr(true),
				},
			},
			out: errors.ErrOverwriteAndNilSource,
		},
		{
			in: File{
				Node: Node{
					Overwrite: util.BoolToPtr(true),
				},
				FileEmbedded1: FileEmbedded1{
					Contents: FileContents{
						Source: util.StrToPtr(""),
					},
				},
			},
			out: nil,
		},
		{
			in: File{
				Node: Node{
					Overwrite: util.BoolToPtr(true),
				},
				FileEmbedded1: FileEmbedded1{
					Contents: FileContents{
						Source: util.StrToPtr("http://example.com"),
					},
				},
			},
			out: nil,
		},
	}

	for i, test := range tests {
		r := test.in.ValidateOverwrite()
		expected := report.ReportFromError(test.out, report.EntryError)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad error: want %v, got %v", i, expected, r)
		}
	}
}

func TestFileContentsValidate(t *testing.T) {
	tests := []struct {
		in  FileContents
		out error
	}{
		{
			in:  FileContents{},
			out: nil,
		},
		{
			in: FileContents{
				Source: util.StrToPtr(""),
			},
			out: nil,
		},
		{
			in: FileContents{
				Source: util.StrToPtr(""),
				Verification: Verification{
					Hash: util.StrToPtr(""),
				},
			},
			out: nil,
		},
		{
			in: FileContents{
				Verification: Verification{
					Hash: util.StrToPtr(""),
				},
			},
			out: errors.ErrVerificationAndNilSource,
		},
	}

	for i, test := range tests {
		r := test.in.ValidateVerification()
		expected := report.ReportFromError(test.out, report.EntryError)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad error: want %v, got %v", i, expected, r)
		}
	}
}
