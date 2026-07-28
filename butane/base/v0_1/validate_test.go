// Copyright 2019 Red Hat, Inc
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

package v0_1

import (
	"fmt"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	"github.com/coreos/butane/config/common"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

// TestValidateFileContents tests that multiple sources (i.e. urls and inline) are not allowed but zero or one sources are
func TestValidateFileContents(t *testing.T) {
	tests := []struct {
		in  FileContents
		out error
	}{
		{},
		{
			// contains invalid (by the validator's definition) combinations of fields,
			// but the translator doesn't care and we can check they all get translated at once
			FileContents{
				Source:      util.StrToPtr("http://example/com"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			nil,
		},
		{
			FileContents{
				Inline:      util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			nil,
		},
		{
			FileContents{
				Source:      util.StrToPtr("data:,hello"),
				Inline:      util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			common.ErrTooManyResourceSources,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			// hardcode inline for now since that's the only place errors occur. Move into the
			// test struct once there's more than one place
			expected.AddOnError(path.New("yaml", "inline"), test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

func TestValidateFileMode(t *testing.T) {
	fileTests := []struct {
		in  File
		out error
	}{
		{
			in:  File{},
			out: nil,
		},
		{
			in: File{
				Mode: util.IntToPtr(0600),
			},
			out: nil,
		},
		{
			in: File{
				Mode: util.IntToPtr(600),
			},
			out: common.ErrDecimalMode,
		},
	}

	for i, test := range fileTests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnWarn(path.New("yaml", "mode"), test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

func TestValidateDirMode(t *testing.T) {
	dirTests := []struct {
		in  Directory
		out error
	}{
		{
			in:  Directory{},
			out: nil,
		},
		{
			in: Directory{
				Mode: util.IntToPtr(01770),
			},
			out: nil,
		},
		{
			in: Directory{
				Mode: util.IntToPtr(1770),
			},
			out: common.ErrDecimalMode,
		},
	}

	for i, test := range dirTests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnWarn(path.New("yaml", "mode"), test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}
