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

package v0_5

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

// TestValidateResource tests that multiple sources (i.e. urls and inline) are not allowed but zero or one sources are
func TestValidateResource(t *testing.T) {
	tests := []struct {
		in      Resource
		out     error
		errPath path.ContextPath
	}{
		{},
		// source specified
		{
			// contains invalid (by the validator's definition) combinations of fields,
			// but the translator doesn't care and we can check they all get translated at once
			Resource{
				Source:      util.StrToPtr("http://example/com"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			nil,
			path.New("yaml"),
		},
		// inline specified
		{
			Resource{
				Inline:      util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			nil,
			path.New("yaml"),
		},
		// local specified
		{
			Resource{
				Local:       util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			nil,
			path.New("yaml"),
		},
		// source + inline, invalid
		{
			Resource{
				Source:      util.StrToPtr("data:,hello"),
				Inline:      util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			common.ErrTooManyResourceSources,
			path.New("yaml", "source"),
		},
		// source + local, invalid
		{
			Resource{
				Source:      util.StrToPtr("data:,hello"),
				Local:       util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			common.ErrTooManyResourceSources,
			path.New("yaml", "source"),
		},
		// inline + local, invalid
		{
			Resource{
				Inline:      util.StrToPtr("hello"),
				Local:       util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			common.ErrTooManyResourceSources,
			path.New("yaml", "inline"),
		},
		// source + inline + local, invalid
		{
			Resource{
				Source:      util.StrToPtr("data:,hello"),
				Inline:      util.StrToPtr("hello"),
				Local:       util.StrToPtr("hello"),
				Compression: util.StrToPtr("gzip"),
				Verification: Verification{
					Hash: util.StrToPtr("this isn't validated"),
				},
			},
			common.ErrTooManyResourceSources,
			path.New("yaml", "source"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

func TestValidateTree(t *testing.T) {
	tests := []struct {
		in  Tree
		out error
	}{
		{
			in:  Tree{},
			out: common.ErrTreeNoLocal,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(path.New("yaml"), test.out)
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

func TestValidateFilesystem(t *testing.T) {
	tests := []struct {
		in      Filesystem
		out     error
		errPath path.ContextPath
	}{
		{
			Filesystem{},
			nil,
			path.New("yaml"),
		},
		{
			Filesystem{
				Device: "/dev/foo",
			},
			nil,
			path.New("yaml"),
		},
		{
			Filesystem{
				Device:        "/dev/foo",
				Format:        util.StrToPtr("zzz"),
				Path:          util.StrToPtr("/z"),
				WithMountUnit: util.BoolToPtr(true),
			},
			nil,
			path.New("yaml"),
		},
		{
			Filesystem{
				Device:        "/dev/foo",
				Format:        util.StrToPtr("swap"),
				WithMountUnit: util.BoolToPtr(true),
			},
			nil,
			path.New("yaml"),
		},
		{
			Filesystem{
				Device:        "/dev/foo",
				WithMountUnit: util.BoolToPtr(true),
			},
			common.ErrMountUnitNoFormat,
			path.New("yaml", "format"),
		},
		{
			Filesystem{
				Device:        "/dev/foo",
				Format:        util.StrToPtr("zzz"),
				WithMountUnit: util.BoolToPtr(true),
			},
			common.ErrMountUnitNoPath,
			path.New("yaml", "path"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

// TestValidateUnit tests that multiple sources (i.e. contents and contents_local) are not allowed but zero or one sources are
func TestValidateUnit(t *testing.T) {
	tests := []struct {
		in      Unit
		out     error
		errPath path.ContextPath
	}{
		{},
		// contents specified
		{
			Unit{
				Contents: util.StrToPtr("hello"),
			},
			nil,
			path.New("yaml"),
		},
		// contents_local specified
		{
			Unit{
				ContentsLocal: util.StrToPtr("hello"),
			},
			nil,
			path.New("yaml"),
		},
		// contents + contents_local, invalid
		{
			Unit{
				Contents:      util.StrToPtr("hello"),
				ContentsLocal: util.StrToPtr("hello, too"),
			},
			common.ErrTooManySystemdSources,
			path.New("yaml", "contents_local"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

// TestValidateDropin tests that multiple sources (i.e. contents and contents_local) are not allowed but zero or one sources are
func TestValidateDropin(t *testing.T) {
	tests := []struct {
		in      Dropin
		out     error
		errPath path.ContextPath
	}{
		{},
		// contents specified
		{
			Dropin{
				Contents: util.StrToPtr("hello"),
			},
			nil,
			path.New("yaml"),
		},
		// contents_local specified
		{
			Dropin{
				ContentsLocal: util.StrToPtr("hello"),
			},
			nil,
			path.New("yaml"),
		},
		// contents + contents_local, invalid
		{
			Dropin{
				Contents:      util.StrToPtr("hello"),
				ContentsLocal: util.StrToPtr("hello, too"),
			},
			common.ErrTooManySystemdSources,
			path.New("yaml", "contents_local"),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate %d", i), func(t *testing.T) {
			actual := test.in.Validate(path.New("yaml"))
			baseutil.VerifyReport(t, test.in, actual)
			expected := report.Report{}
			expected.AddOnError(test.errPath, test.out)
			assert.Equal(t, expected, actual, "bad report")
		})
	}
}

// TestUnkownIgnitionVersion tests that butane will raise a warning but will not fail when an ignition config with an unkown version is specified
func TestUnkownIgnitionVersion(t *testing.T) {
	test := struct {
		in      Resource
		out     error
		errPath path.ContextPath
	}{
		Resource{
			Inline: util.StrToPtr(`{"ignition": {"version": "10.0.0"}}`),
		},
		common.ErrUnkownIgnitionVersion,
		path.New("yaml", "ignition", "config", "version"),
	}
	path := path.New("yaml", "ignition", "config")
	// Skipping baseutil.VerifyReport because it expects all referenced paths to exist in the struct.
	// In this test, "ignition.config" doesn't exist, so VerifyReport would fail. However, we still need
	// to pass this path to Validate() to trigger the unknown Ignition version warning we're testing for.
	actual := test.in.Validate(path)
	expected := report.Report{}
	expected.AddOnWarn(test.errPath, test.out)
	assert.Equal(t, expected, actual, "bad report")
}
