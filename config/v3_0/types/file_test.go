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
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
)

func TestFileValidateOverwrite(t *testing.T) {
	tests := []struct {
		in  File
		out error
	}{
		{
			File{},
			nil,
		},
		{
			File{
				Node: Node{
					Overwrite: util.BoolToPtr(true),
				},
			},
			errors.ErrOverwriteAndNilSource,
		},
		{
			File{
				Node: Node{
					Overwrite: util.BoolToPtr(true),
				},
				FileEmbedded1: FileEmbedded1{
					Contents: FileContents{
						Source: util.StrToPtr(""),
					},
				},
			},
			nil,
		},
		{
			File{
				Node: Node{
					Overwrite: util.BoolToPtr(true),
				},
				FileEmbedded1: FileEmbedded1{
					Contents: FileContents{
						Source: util.StrToPtr("http://example.com"),
					},
				},
			},
			nil,
		},
	}

	for i, test := range tests {
		err := test.in.validateOverwrite()
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
		}
	}
}

func TestFileContentsValidate(t *testing.T) {
	tests := []struct {
		in  FileContents
		out error
	}{
		{
			FileContents{},
			nil,
		},
		{
			FileContents{
				Source: util.StrToPtr(""),
			},
			nil,
		},
		{
			FileContents{
				Source: util.StrToPtr(""),
				Verification: Verification{
					Hash: util.StrToPtr(""),
				},
			},
			nil,
		},
		{
			FileContents{
				Verification: Verification{
					Hash: util.StrToPtr(""),
				},
			},
			errors.ErrVerificationAndNilSource,
		},
	}

	for i, test := range tests {
		err := test.in.validateVerification()
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
		}
	}
}
