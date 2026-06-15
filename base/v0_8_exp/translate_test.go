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

package v0_8_exp

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	"github.com/coreos/butane/config/common"
	confutil "github.com/coreos/butane/config/util"
	"github.com/coreos/butane/translate"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

// Most of this is covered by the Ignition translator generic tests, so just test the custom bits

var (
	osStatName string
	osNotFound string
)

func init() {
	if runtime.GOOS == "windows" {
		osStatName = "GetFileAttributesEx"
		osNotFound = "The system cannot find the file specified."
	} else {
		osStatName = "stat"
		osNotFound = "no such file or directory"
	}
}

// TestTranslateFile tests translating the ct storage.files.[i] entries to ignition storage.files.[i] entries.
func TestTranslateFile(t *testing.T) {
	zzz := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	zzzURI, zzzCompression := baseutil.CompressDataURL(t, []byte(zzz))
	random := "\xc0\x9cl\x01\x89i\xa5\xbfW\xe4\x1b\xf4J_\xb79P\xa3#\xa7"
	randomURI, randomCompression := baseutil.CompressDataURL(t, []byte(random))

	filesDir := t.TempDir()
	fileContents := map[string]string{
		"file-1":        "file contents\n",
		"file-2":        zzz,
		"file-3":        random,
		"subdir/file-4": "subdir file contents\n",
	}
	for name, contents := range fileContents {
		if err := os.MkdirAll(filepath.Join(filesDir, filepath.Dir(name)), 0755); err != nil {
			t.Error(err)
			return
		}
		err := os.WriteFile(filepath.Join(filesDir, name), []byte(contents), 0644)
		if err != nil {
			t.Error(err)
			return
		}
	}

	tests := []struct {
		in         File
		out        types.File
		exceptions []translate.Translation
		report     string
		options    common.TranslateOptions
	}{
		{
			File{},
			types.File{},
			nil,
			"",
			common.TranslateOptions{},
		},
		{
			// contains invalid (by the validator's definition) combinations of fields,
			// but the translator doesn't care and we can check they all get translated at once
			File{
				Path: "/foo",
				Group: NodeGroup{
					ID:   util.IntToPtr(1),
					Name: util.StrToPtr("foobar"),
				},
				User: NodeUser{
					ID:   util.IntToPtr(1),
					Name: util.StrToPtr("bazquux"),
				},
				Mode: util.IntToPtr(420),
				Append: []Resource{
					{
						Source:      util.StrToPtr("http://example/com"),
						Compression: util.StrToPtr("gzip"),
						HTTPHeaders: HTTPHeaders{
							HTTPHeader{
								Name:  "Header",
								Value: util.StrToPtr("this isn't validated"),
							},
						},
						Verification: Verification{
							Hash: util.StrToPtr("this isn't validated"),
						},
					},
					{
						Inline:      util.StrToPtr("hello"),
						Compression: util.StrToPtr("gzip"),
						HTTPHeaders: HTTPHeaders{
							HTTPHeader{
								Name:  "Header",
								Value: util.StrToPtr("this isn't validated"),
							},
						},
						Verification: Verification{
							Hash: util.StrToPtr("this isn't validated"),
						},
					},
					{
						Local: util.StrToPtr("file-1"),
					},
				},
				Overwrite: util.BoolToPtr(true),
				Contents: Resource{
					Source:      util.StrToPtr("http://example/com"),
					Compression: util.StrToPtr("gzip"),
					HTTPHeaders: HTTPHeaders{
						HTTPHeader{
							Name:  "Header",
							Value: util.StrToPtr("this isn't validated"),
						},
					},
					Verification: Verification{
						Hash: util.StrToPtr("this isn't validated"),
					},
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
					Group: types.NodeGroup{
						ID:   util.IntToPtr(1),
						Name: util.StrToPtr("foobar"),
					},
					User: types.NodeUser{
						ID:   util.IntToPtr(1),
						Name: util.StrToPtr("bazquux"),
					},
					Overwrite: util.BoolToPtr(true),
				},
				FileEmbedded1: types.FileEmbedded1{
					Mode: util.IntToPtr(420),
					Append: []types.Resource{
						{
							Source:      util.StrToPtr("http://example/com"),
							Compression: util.StrToPtr("gzip"),
							HTTPHeaders: types.HTTPHeaders{
								types.HTTPHeader{
									Name:  "Header",
									Value: util.StrToPtr("this isn't validated"),
								},
							},
							Verification: types.Verification{
								Hash: util.StrToPtr("this isn't validated"),
							},
						},
						{
							Source:      util.StrToPtr("data:,hello"),
							Compression: util.StrToPtr("gzip"),
							HTTPHeaders: types.HTTPHeaders{
								types.HTTPHeader{
									Name:  "Header",
									Value: util.StrToPtr("this isn't validated"),
								},
							},
							Verification: types.Verification{
								Hash: util.StrToPtr("this isn't validated"),
							},
						},
						{
							Source:      util.StrToPtr("data:,file%20contents%0A"),
							Compression: util.StrToPtr(""),
						},
					},
					Contents: types.Resource{
						Source:      util.StrToPtr("http://example/com"),
						Compression: util.StrToPtr("gzip"),
						HTTPHeaders: types.HTTPHeaders{
							types.HTTPHeader{
								Name:  "Header",
								Value: util.StrToPtr("this isn't validated"),
							},
						},
						Verification: types.Verification{
							Hash: util.StrToPtr("this isn't validated"),
						},
					},
				},
			},
			[]translate.Translation{
				{
					From: path.New("yaml", "append", 0, "http_headers"),
					To:   path.New("json", "append", 0, "httpHeaders"),
				},
				{
					From: path.New("yaml", "append", 0, "http_headers", 0),
					To:   path.New("json", "append", 0, "httpHeaders", 0),
				},
				{
					From: path.New("yaml", "append", 0, "http_headers", 0, "name"),
					To:   path.New("json", "append", 0, "httpHeaders", 0, "name"),
				},
				{
					From: path.New("yaml", "append", 0, "http_headers", 0, "value"),
					To:   path.New("json", "append", 0, "httpHeaders", 0, "value"),
				},
				{
					From: path.New("yaml", "append", 1, "inline"),
					To:   path.New("json", "append", 1, "source"),
				},
				{
					From: path.New("yaml", "append", 1, "http_headers"),
					To:   path.New("json", "append", 1, "httpHeaders"),
				},
				{
					From: path.New("yaml", "append", 1, "http_headers", 0),
					To:   path.New("json", "append", 1, "httpHeaders", 0),
				},
				{
					From: path.New("yaml", "append", 1, "http_headers", 0, "name"),
					To:   path.New("json", "append", 1, "httpHeaders", 0, "name"),
				},
				{
					From: path.New("yaml", "append", 1, "http_headers", 0, "value"),
					To:   path.New("json", "append", 1, "httpHeaders", 0, "value"),
				},
				{
					From: path.New("yaml", "append", 2, "local"),
					To:   path.New("json", "append", 2, "source"),
				},
				{
					From: path.New("yaml", "append", 2, "local"),
					To:   path.New("json", "append", 2, "compression"),
				},
				{
					From: path.New("yaml", "contents", "http_headers"),
					To:   path.New("json", "contents", "httpHeaders"),
				},
				{
					From: path.New("yaml", "contents", "http_headers", 0),
					To:   path.New("json", "contents", "httpHeaders", 0),
				},
				{
					From: path.New("yaml", "contents", "http_headers", 0, "name"),
					To:   path.New("json", "contents", "httpHeaders", 0, "name"),
				},
				{
					From: path.New("yaml", "contents", "http_headers", 0, "value"),
					To:   path.New("json", "contents", "httpHeaders", 0, "value"),
				},
			},
			"",
			common.TranslateOptions{
				FilesDir: filesDir,
			},
		},
		// inline file contents
		{
			File{
				Path: "/foo",
				Contents: Resource{
					// String is too short for auto gzip compression
					Inline: util.StrToPtr("xyzzy"),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
				FileEmbedded1: types.FileEmbedded1{
					Contents: types.Resource{
						Source:      util.StrToPtr("data:,xyzzy"),
						Compression: util.StrToPtr(""),
					},
				},
			},
			[]translate.Translation{
				{
					From: path.New("yaml", "contents", "inline"),
					To:   path.New("json", "contents", "source"),
				},
				{
					From: path.New("yaml", "contents", "inline"),
					To:   path.New("json", "contents", "compression"),
				},
			},
			"",
			common.TranslateOptions{},
		},
		// local file contents
		{
			File{
				Path: "/foo",
				Contents: Resource{
					Local: util.StrToPtr("file-1"),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
				FileEmbedded1: types.FileEmbedded1{
					Contents: types.Resource{
						Source:      util.StrToPtr("data:,file%20contents%0A"),
						Compression: util.StrToPtr(""),
					},
				},
			},
			[]translate.Translation{
				{
					From: path.New("yaml", "contents", "local"),
					To:   path.New("json", "contents", "source"),
				},
				{
					From: path.New("yaml", "contents", "local"),
					To:   path.New("json", "contents", "compression"),
				},
			},
			"",
			common.TranslateOptions{
				FilesDir: filesDir,
			},
		},
		// local file in subdirectory
		{
			File{
				Path: "/foo",
				Contents: Resource{
					Local: util.StrToPtr("subdir/file-4"),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
				FileEmbedded1: types.FileEmbedded1{
					Contents: types.Resource{
						Source:      util.StrToPtr("data:,subdir%20file%20contents%0A"),
						Compression: util.StrToPtr(""),
					},
				},
			},
			[]translate.Translation{
				{
					From: path.New("yaml", "contents", "local"),
					To:   path.New("json", "contents", "source"),
				},
				{
					From: path.New("yaml", "contents", "local"),
					To:   path.New("json", "contents", "compression"),
				},
			},
			"",
			common.TranslateOptions{
				FilesDir: filesDir,
			},
		},
		// filesDir not specified
		{
			File{
				Path: "/foo",
				Contents: Resource{
					Local: util.StrToPtr("file-1"),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
			},
			[]translate.Translation{},
			"error at $.contents.local: " + common.ErrNoFilesDir.Error() + "\n",
			common.TranslateOptions{},
		},
		// attempted directory traversal
		{
			File{
				Path: "/foo",
				Contents: Resource{
					Local: util.StrToPtr("../file-1"),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
			},
			[]translate.Translation{},
			"error at $.contents.local: " + common.ErrFilesDirEscape.Error() + "\n",
			common.TranslateOptions{
				FilesDir: filesDir,
			},
		},
		// attempted inclusion of nonexistent file
		{
			File{
				Path: "/foo",
				Contents: Resource{
					Local: util.StrToPtr("file-missing"),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
			},
			[]translate.Translation{},
			"error at $.contents.local: open " + filepath.Join(filesDir, "file-missing") + ": " + osNotFound + "\n",
			common.TranslateOptions{
				FilesDir: filesDir,
			},
		},
		// inline and local automatic file encoding
		{
			File{
				Path: "/foo",
				Contents: Resource{
					// gzip
					Inline: util.StrToPtr(zzz),
				},
				Append: []Resource{
					{
						// gzip
						Local: util.StrToPtr("file-2"),
					},
					{
						// base64
						Inline: util.StrToPtr(random),
					},
					{
						// base64
						Local: util.StrToPtr("file-3"),
					},
					{
						// URL-escaped
						Inline:      util.StrToPtr(zzz),
						Compression: util.StrToPtr("invalid"),
					},
					{
						// URL-escaped
						Local:       util.StrToPtr("file-2"),
						Compression: util.StrToPtr("invalid"),
					},
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
				FileEmbedded1: types.FileEmbedded1{
					Contents: types.Resource{
						Source:      util.StrToPtr(zzzURI),
						Compression: util.StrToPtr(zzzCompression),
					},
					Append: []types.Resource{
						{
							Source:      util.StrToPtr(zzzURI),
							Compression: util.StrToPtr(zzzCompression),
						},
						{
							Source:      util.StrToPtr(randomURI),
							Compression: util.StrToPtr(randomCompression),
						},
						{
							Source:      util.StrToPtr(randomURI),
							Compression: util.StrToPtr(randomCompression),
						},
						{
							Source:      util.StrToPtr("data:," + zzz),
							Compression: util.StrToPtr("invalid"),
						},
						{
							Source:      util.StrToPtr("data:," + zzz),
							Compression: util.StrToPtr("invalid"),
						},
					},
				},
			},
			[]translate.Translation{
				{
					From: path.New("yaml", "contents", "inline"),
					To:   path.New("json", "contents", "source"),
				},
				{
					From: path.New("yaml", "contents", "inline"),
					To:   path.New("json", "contents", "compression"),
				},
				{
					From: path.New("yaml", "append", 0, "local"),
					To:   path.New("json", "append", 0, "source"),
				},
				{
					From: path.New("yaml", "append", 0, "local"),
					To:   path.New("json", "append", 0, "compression"),
				},
				{
					From: path.New("yaml", "append", 1, "inline"),
					To:   path.New("json", "append", 1, "source"),
				},
				{
					From: path.New("yaml", "append", 1, "inline"),
					To:   path.New("json", "append", 1, "compression"),
				},
				{
					From: path.New("yaml", "append", 2, "local"),
					To:   path.New("json", "append", 2, "source"),
				},
				{
					From: path.New("yaml", "append", 2, "local"),
					To:   path.New("json", "append", 2, "compression"),
				},
				{
					From: path.New("yaml", "append", 3, "inline"),
					To:   path.New("json", "append", 3, "source"),
				},
				{
					From: path.New("yaml", "append", 4, "local"),
					To:   path.New("json", "append", 4, "source"),
				},
			},
			"",
			common.TranslateOptions{
				FilesDir: filesDir,
			},
		},
		// Test disable automatic gzip compression
		{
			File{
				Path: "/foo",
				Contents: Resource{
					Inline: util.StrToPtr(zzz),
				},
			},
			types.File{
				Node: types.Node{
					Path: "/foo",
				},
				FileEmbedded1: types.FileEmbedded1{
					Contents: types.Resource{
						Source:      util.StrToPtr("data:," + zzz),
						Compression: util.StrToPtr(""),
					},
				},
			},
			[]translate.Translation{
				{
					From: path.New("yaml", "contents", "inline"),
					To:   path.New("json", "contents", "source"),
				},
				{
					From: path.New("yaml", "contents", "inline"),
					To:   path.New("json", "contents", "compression"),
				},
			},
			"",
			common.TranslateOptions{
				NoResourceAutoCompression: true,
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := translateFile(test.in, test.options)
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, test.report, r.String(), "bad report")
			baseutil.VerifyTranslations(t, translations, test.exceptions)
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateDirectory tests translating the ct storage.directories.[i] entries to ignition storage.directories.[i] entires.
func TestTranslateDirectory(t *testing.T) {
	tests := []struct {
		in  Directory
		out types.Directory
	}{
		{
			Directory{},
			types.Directory{},
		},
		{
			// contains invalid (by the validator's definition) combinations of fields,
			// but the translator doesn't care and we can check they all get translated at once
			Directory{
				Path: "/foo",
				Group: NodeGroup{
					ID:   util.IntToPtr(1),
					Name: util.StrToPtr("foobar"),
				},
				User: NodeUser{
					ID:   util.IntToPtr(1),
					Name: util.StrToPtr("bazquux"),
				},
				Mode:      util.IntToPtr(420),
				Overwrite: util.BoolToPtr(true),
			},
			types.Directory{
				Node: types.Node{
					Path: "/foo",
					Group: types.NodeGroup{
						ID:   util.IntToPtr(1),
						Name: util.StrToPtr("foobar"),
					},
					User: types.NodeUser{
						ID:   util.IntToPtr(1),
						Name: util.StrToPtr("bazquux"),
					},
					Overwrite: util.BoolToPtr(true),
				},
				DirectoryEmbedded1: types.DirectoryEmbedded1{
					Mode: util.IntToPtr(420),
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := translateDirectory(test.in, common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateLink tests translating the ct storage.links.[i] entries to ignition storage.links.[i] entires.
func TestTranslateLink(t *testing.T) {
	tests := []struct {
		in  Link
		out types.Link
	}{
		{
			Link{},
			types.Link{},
		},
		{
			// contains invalid (by the validator's definition) combinations of fields,
			// but the translator doesn't care and we can check they all get translated at once
			Link{
				Path: "/foo",
				Group: NodeGroup{
					ID:   util.IntToPtr(1),
					Name: util.StrToPtr("foobar"),
				},
				User: NodeUser{
					ID:   util.IntToPtr(1),
					Name: util.StrToPtr("bazquux"),
				},
				Overwrite: util.BoolToPtr(true),
				Target:    util.StrToPtr("/bar"),
				Hard:      util.BoolToPtr(false),
			},
			types.Link{
				Node: types.Node{
					Path: "/foo",
					Group: types.NodeGroup{
						ID:   util.IntToPtr(1),
						Name: util.StrToPtr("foobar"),
					},
					User: types.NodeUser{
						ID:   util.IntToPtr(1),
						Name: util.StrToPtr("bazquux"),
					},
					Overwrite: util.BoolToPtr(true),
				},
				LinkEmbedded1: types.LinkEmbedded1{
					Target: util.StrToPtr("/bar"),
					Hard:   util.BoolToPtr(false),
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := translateLink(test.in, common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateFilesystem tests translating the butane storage.filesystems.[i] entries to ignition storage.filesystems.[i] entries.
func TestTranslateFilesystem(t *testing.T) {
	tests := []struct {
		in  Filesystem
		out types.Filesystem
	}{
		{
			Filesystem{},
			types.Filesystem{},
		},
		{
			// contains invalid (by the validator's definition) combinations of fields,
			// but the translator doesn't care and we can check they all get translated at once
			Filesystem{
				Device:         "/foo",
				Format:         util.StrToPtr("/bar"),
				Label:          util.StrToPtr("/baz"),
				MountOptions:   []string{"yes", "no", "maybe"},
				Options:        []string{"foo", "foo", "bar"},
				Path:           util.StrToPtr("/quux"),
				UUID:           util.StrToPtr("1234"),
				WipeFilesystem: util.BoolToPtr(true),
				WithMountUnit:  util.BoolToPtr(true),
			},
			types.Filesystem{
				Device:         "/foo",
				Format:         util.StrToPtr("/bar"),
				Label:          util.StrToPtr("/baz"),
				MountOptions:   []types.MountOption{"yes", "no", "maybe"},
				Options:        []types.FilesystemOption{"foo", "foo", "bar"},
				Path:           util.StrToPtr("/quux"),
				UUID:           util.StrToPtr("1234"),
				WipeFilesystem: util.BoolToPtr(true),
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			// Filesystem doesn't have a custom translator, so embed in a
			// complete config
			in := Config{
				Storage: Storage{
					Filesystems: []Filesystem{test.in},
				},
			}
			expected := []types.Filesystem{test.out}
			actual, translations, r := in.ToIgn3_7Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, expected, actual.Storage.Filesystems, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			// FIXME: Zero values are pruned from merge transcripts and
			// TranslationSets to make them more compact in debug output
			// and tests.  As a result, if the user specifies an empty
			// struct in a list, the translation coverage will be
			// incomplete and the report entry marker will end up
			// pointing to the base of the list, or to a parent if the
			// struct is the only entry in the list.  Skip the coverage
			// test for this case.
			if !reflect.ValueOf(test.out).IsZero() {
				assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
			}
		})
	}
}

// TestTranslateMountUnit tests the Butane storage.filesystems.[i].with_mount_unit flag.
func TestTranslateMountUnit(t *testing.T) {
	tests := []struct {
		in  Config
		out types.Config
	}{
		// local mount with options, overridden enabled flag
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/disk/by-label/foo",
							Format:        util.StrToPtr("ext4"),
							MountOptions:  []string{"ro", "noatime"},
							Path:          util.StrToPtr("/var/lib/containers"),
							WithMountUnit: util.BoolToPtr(true),
						},
					},
				},
				Systemd: Systemd{
					Units: []Unit{
						{
							Name:    "var-lib-containers.mount",
							Enabled: util.BoolToPtr(false),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device:       "/dev/disk/by-label/foo",
							Format:       util.StrToPtr("ext4"),
							MountOptions: []types.MountOption{"ro", "noatime"},
							Path:         util.StrToPtr("/var/lib/containers"),
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled: util.BoolToPtr(false),
							Contents: util.StrToPtr(`# Generated by Butane
[Unit]
Requires=systemd-fsck@dev-disk-by\x2dlabel-foo.service
After=systemd-fsck@dev-disk-by\x2dlabel-foo.service

[Mount]
Where=/var/lib/containers
What=/dev/disk/by-label/foo
Type=ext4
Options=ro,noatime

[Install]
RequiredBy=local-fs.target`),
							Name: "var-lib-containers.mount",
						},
					},
				},
			},
		},
		// remote mount with options
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/mapper/foo-bar",
							Format:        util.StrToPtr("ext4"),
							MountOptions:  []string{"ro", "noatime"},
							Path:          util.StrToPtr("/var/lib/containers"),
							WithMountUnit: util.BoolToPtr(true),
						},
					},
					Luks: []Luks{
						{
							Name:   "foo-bar",
							Device: util.StrToPtr("/dev/bar"),
							Clevis: Clevis{
								Tang: []Tang{
									{
										URL: "http://example.com",
									},
								},
							},
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device:       "/dev/mapper/foo-bar",
							Format:       util.StrToPtr("ext4"),
							MountOptions: []types.MountOption{"ro", "noatime"},
							Path:         util.StrToPtr("/var/lib/containers"),
						},
					},
					Luks: []types.Luks{
						{
							Name:   "foo-bar",
							Device: util.StrToPtr("/dev/bar"),
							Clevis: types.Clevis{
								Tang: []types.Tang{
									{
										URL: "http://example.com",
									},
								},
							},
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled: util.BoolToPtr(true),
							Contents: util.StrToPtr(`# Generated by Butane
[Unit]
Requires=systemd-fsck@dev-mapper-foo\x2dbar.service
After=systemd-fsck@dev-mapper-foo\x2dbar.service

[Mount]
Where=/var/lib/containers
What=/dev/mapper/foo-bar
Type=ext4
Options=ro,noatime,_netdev

[Install]
RequiredBy=remote-fs.target`),
							Name: "var-lib-containers.mount",
						},
					},
				},
			},
		},
		// local mount, no options
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/disk/by-label/foo",
							Format:        util.StrToPtr("ext4"),
							Path:          util.StrToPtr("/var/lib/containers"),
							WithMountUnit: util.BoolToPtr(true),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/disk/by-label/foo",
							Format: util.StrToPtr("ext4"),
							Path:   util.StrToPtr("/var/lib/containers"),
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled: util.BoolToPtr(true),
							Contents: util.StrToPtr(`# Generated by Butane
[Unit]
Requires=systemd-fsck@dev-disk-by\x2dlabel-foo.service
After=systemd-fsck@dev-disk-by\x2dlabel-foo.service

[Mount]
Where=/var/lib/containers
What=/dev/disk/by-label/foo
Type=ext4

[Install]
RequiredBy=local-fs.target`),
							Name: "var-lib-containers.mount",
						},
					},
				},
			},
		},
		// remote mount, no options
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/mapper/foo-bar",
							Format:        util.StrToPtr("ext4"),
							Path:          util.StrToPtr("/var/lib/containers"),
							WithMountUnit: util.BoolToPtr(true),
						},
					},
					Luks: []Luks{
						{
							Name:   "foo-bar",
							Device: util.StrToPtr("/dev/bar"),
							Clevis: Clevis{
								Tang: []Tang{
									{
										URL: "http://example.com",
									},
								},
							},
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/mapper/foo-bar",
							Format: util.StrToPtr("ext4"),
							Path:   util.StrToPtr("/var/lib/containers"),
						},
					},
					Luks: []types.Luks{
						{
							Name:   "foo-bar",
							Device: util.StrToPtr("/dev/bar"),
							Clevis: types.Clevis{
								Tang: []types.Tang{
									{
										URL: "http://example.com",
									},
								},
							},
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled: util.BoolToPtr(true),
							Contents: util.StrToPtr(`# Generated by Butane
[Unit]
Requires=systemd-fsck@dev-mapper-foo\x2dbar.service
After=systemd-fsck@dev-mapper-foo\x2dbar.service

[Mount]
Where=/var/lib/containers
What=/dev/mapper/foo-bar
Type=ext4
Options=_netdev

[Install]
RequiredBy=remote-fs.target`),
							Name: "var-lib-containers.mount",
						},
					},
				},
			},
		},
		// overridden mount unit
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/disk/by-label/foo",
							Format:        util.StrToPtr("ext4"),
							Path:          util.StrToPtr("/var/lib/containers"),
							WithMountUnit: util.BoolToPtr(true),
						},
					},
				},
				Systemd: Systemd{
					Units: []Unit{
						{
							Name:     "var-lib-containers.mount",
							Contents: util.StrToPtr("[Service]\nExecStart=/bin/false\n"),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/disk/by-label/foo",
							Format: util.StrToPtr("ext4"),
							Path:   util.StrToPtr("/var/lib/containers"),
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled:  util.BoolToPtr(true),
							Contents: util.StrToPtr("[Service]\nExecStart=/bin/false\n"),
							Name:     "var-lib-containers.mount",
						},
					},
				},
			},
		},
		// swap, no options
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/disk/by-label/foo",
							Format:        util.StrToPtr("swap"),
							WithMountUnit: util.BoolToPtr(true),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/disk/by-label/foo",
							Format: util.StrToPtr("swap"),
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled: util.BoolToPtr(true),
							Contents: util.StrToPtr(`# Generated by Butane
[Swap]
What=/dev/disk/by-label/foo

[Install]
RequiredBy=swap.target`),
							Name: "dev-disk-by\\x2dlabel-foo.swap",
						},
					},
				},
			},
		},
		// swap with options
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device:        "/dev/disk/by-label/foo",
							Format:        util.StrToPtr("swap"),
							MountOptions:  []string{"pri=1", "discard=pages"},
							WithMountUnit: util.BoolToPtr(true),
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device:       "/dev/disk/by-label/foo",
							Format:       util.StrToPtr("swap"),
							MountOptions: []types.MountOption{"pri=1", "discard=pages"},
						},
					},
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Enabled: util.BoolToPtr(true),
							Contents: util.StrToPtr(`# Generated by Butane
[Swap]
What=/dev/disk/by-label/foo
Options=pri=1,discard=pages

[Install]
RequiredBy=swap.target`),
							Name: "dev-disk-by\\x2dlabel-foo.swap",
						},
					},
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			out, translations, r := test.in.ToIgn3_7Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, out, "bad output")
			assert.Equal(t, report.Report{}, r, "expected empty report")
			assert.NoError(t, translations.DebugVerifyCoverage(out), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateTree tests translating the butane storage.trees.[i] entries to ignition storage.files.[i] entries.
func TestTranslateTree(t *testing.T) {
	deepPath := "tree/subdir/subdir/subdir/subdir/subdir/subdir/subdir/subdir/subdir/file"
	deepPathURI, deepPathCompression := baseutil.CompressDataURL(t, []byte(deepPath))

	tests := []struct {
		options    *common.TranslateOptions // defaulted if not specified
		dirDirs    map[string]os.FileMode   // relative path -> mode
		dirFiles   map[string]os.FileMode   // relative path -> mode
		dirLinks   map[string]string        // relative path -> target
		dirSockets []string                 // relative path
		inTrees    []Tree
		inFiles    []File
		inDirs     []Directory
		inLinks    []Link
		outFiles   []types.File
		outDirs    []types.Directory
		outLinks   []types.Link
		report     string
		skip       func(t *testing.T)
	}{
		// smoke test
		{},
		// basic functionality
		{
			dirFiles: map[string]os.FileMode{
				"tree/executable":            0700,
				"tree/file":                  0600,
				"tree/overridden":            0644,
				"tree/overridden-executable": 0700,
				"tree/subdir/file":           0644,
				// compressed contents
				"tree/subdir/subdir/subdir/subdir/subdir/subdir/subdir/subdir/subdir/file": 0644,
				"tree2/file": 0600,
			},
			dirLinks: map[string]string{
				"tree/subdir/bad-link":        "../nonexistent",
				"tree/subdir/link":            "../file",
				"tree/subdir/overridden-link": "../file",
			},
			inTrees: []Tree{
				{
					Local: "tree",
				},
				{
					Local: "tree2",
					Path:  util.StrToPtr("/etc"),
				},
			},
			inFiles: []File{
				{
					Path: "/overridden",
					Mode: util.IntToPtr(0600),
					User: NodeUser{
						Name: util.StrToPtr("bovik"),
					},
				},
				{
					Path: "/overridden-executable",
					Mode: util.IntToPtr(0600),
					User: NodeUser{
						Name: util.StrToPtr("bovik"),
					},
				},
			},
			inLinks: []Link{
				{
					Path: "/subdir/overridden-link",
					User: NodeUser{
						Name: util.StrToPtr("bovik"),
					},
				},
			},
			outFiles: []types.File{
				{
					Node: types.Node{
						Path: "/overridden",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Foverridden"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0600),
					},
				},
				{
					Node: types.Node{
						Path: "/overridden-executable",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Foverridden-executable"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0600),
					},
				},
				{
					Node: types.Node{
						Path: "/executable",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Fexecutable"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(func() int {
							if runtime.GOOS != "windows" {
								return 0755
							} else {
								// Windows doesn't have executable bits
								return 0644
							}
						}()),
					},
				},
				{
					Node: types.Node{
						Path: "/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0644),
					},
				},
				{
					Node: types.Node{
						Path: "/subdir/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Fsubdir%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0644),
					},
				},
				{
					Node: types.Node{
						Path: "/subdir/subdir/subdir/subdir/subdir/subdir/subdir/subdir/subdir/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr(deepPathURI),
							Compression: util.StrToPtr(deepPathCompression),
						},
						Mode: util.IntToPtr(0644),
					},
				},
				{
					Node: types.Node{
						Path: "/etc/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree2%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0644),
					},
				},
			},
			outLinks: []types.Link{
				{
					Node: types.Node{
						Path: "/subdir/overridden-link",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
					},
					LinkEmbedded1: types.LinkEmbedded1{
						Target: util.StrToPtr("../file"),
					},
				},
				{
					Node: types.Node{
						Path: "/subdir/bad-link",
					},
					LinkEmbedded1: types.LinkEmbedded1{
						Target: util.StrToPtr("../nonexistent"),
					},
				},
				{
					Node: types.Node{
						Path: "/subdir/link",
					},
					LinkEmbedded1: types.LinkEmbedded1{
						Target: util.StrToPtr("../file"),
					},
				},
			},
		},
		// TranslationSet completeness without overrides
		{
			dirFiles: map[string]os.FileMode{
				"tree/file":        0600,
				"tree/subdir/file": 0644,
			},
			dirDirs: map[string]os.FileMode{
				"tree/dir": 0700,
			},
			dirLinks: map[string]string{
				"tree/subdir/link": "../file",
			},
			inTrees: []Tree{
				{
					Local: "tree",
				},
			},
			outFiles: []types.File{
				{
					Node: types.Node{
						Path: "/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0644),
					},
				},
				{
					Node: types.Node{
						Path: "/subdir/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Fsubdir%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0644),
					},
				},
			},
			outLinks: []types.Link{
				{
					Node: types.Node{
						Path: "/subdir/link",
					},
					LinkEmbedded1: types.LinkEmbedded1{
						Target: util.StrToPtr("../file"),
					},
				},
			},
		},
		// collisions
		{
			dirFiles: map[string]os.FileMode{
				"tree0/file":         0600,
				"tree1/directory":    0600,
				"tree2/link":         0600,
				"tree3/file-partial": 0600, // should be okay
				"tree4/link-partial": 0600,
				"tree5/tree-file":    0600, // set up for tree/tree collision
				"tree6/tree-file":    0600,
				"tree15/tree-link":   0600,
			},
			dirLinks: map[string]string{
				"tree7/file":          "file",
				"tree8/directory":     "file",
				"tree9/link":          "file",
				"tree10/file-partial": "file",
				"tree11/link-partial": "file", // should be okay
				"tree12/tree-file":    "file",
				"tree13/tree-link":    "file", // set up for tree/tree collision
				"tree14/tree-link":    "file",
			},
			inTrees: []Tree{
				{
					Local: "tree0",
				},
				{
					Local: "tree1",
				},
				{
					Local: "tree2",
				},
				{
					Local: "tree3",
				},
				{
					Local: "tree4",
				},
				{
					Local: "tree5",
				},
				{
					Local: "tree6",
				},
				{
					Local: "tree7",
				},
				{
					Local: "tree8",
				},
				{
					Local: "tree9",
				},
				{
					Local: "tree10",
				},
				{
					Local: "tree11",
				},
				{
					Local: "tree12",
				},
				{
					Local: "tree13",
				},
				{
					Local: "tree14",
				},
				{
					Local: "tree15",
				},
			},
			inFiles: []File{
				{
					Path: "/file",
					Contents: Resource{
						Source: util.StrToPtr("data:,foo"),
					},
				},
				{
					Path: "/file-partial",
				},
			},
			inDirs: []Directory{
				{
					Path: "/directory",
				},
			},
			inLinks: []Link{
				{
					Path:   "/link",
					Target: util.StrToPtr("file"),
				},
				{
					Path: "/link-partial",
				},
			},
			report: "error at $.storage.trees.0: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.1: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.2: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.4: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.6: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.7: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.8: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.9: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.10: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.12: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.14: " + common.ErrNodeExists.Error() + "\n" +
				"error at $.storage.trees.15: " + common.ErrNodeExists.Error() + "\n",
		},
		// files-dir escape
		{
			inTrees: []Tree{
				{
					Local: "../escape",
				},
			},
			report: "error at $.storage.trees.0: " + common.ErrFilesDirEscape.Error() + "\n",
		},
		// no files-dir
		{
			options: &common.TranslateOptions{},
			inTrees: []Tree{
				{
					Local: "tree",
				},
			},
			report: "error at $.storage.trees.0: " + common.ErrNoFilesDir.Error() + "\n",
		},
		// non-file/dir/symlink in directory tree
		{
			dirSockets: []string{
				"tree/socket",
			},
			inTrees: []Tree{
				{
					Local: "tree",
				},
			},
			report: "error at $.storage.trees.0: " + common.ErrFileType.Error() + "\n",
			skip: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					// Windows supports Unix domain sockets, but os.Stat()
					// doesn't detect them correctly.
					t.Skip("skipping test due to https://github.com/golang/go/issues/33357")
				}
			},
		},
		// unreadable file
		{
			dirDirs: map[string]os.FileMode{
				"tree/subdir": 0000,
				"tree2":       0000,
			},
			dirFiles: map[string]os.FileMode{
				"tree/file": 0000,
			},
			inTrees: []Tree{
				{
					Local: "tree",
				},
				{
					Local: "tree2",
				},
			},
			report: "error at $.storage.trees.0: open %FilesDir%/tree/file: permission denied\n" +
				"error at $.storage.trees.0: open %FilesDir%/tree/subdir: permission denied\n" +
				"error at $.storage.trees.1: open %FilesDir%/tree2: permission denied\n",
			skip: func(t *testing.T) {
				if runtime.GOOS == "windows" {
					// os.Chmod() only respects the writable bit and there
					// isn't a trivial way to make inodes inaccessible
					t.Skip("skipping test on Windows")
				}
			},
		},
		// local is not a directory
		{
			dirFiles: map[string]os.FileMode{
				"tree": 0600,
			},
			inTrees: []Tree{
				{
					Local: "tree",
				},
				{
					Local: "nonexistent",
				},
			},
			report: "error at $.storage.trees.0: " + common.ErrTreeNotDirectory.Error() + "\n" +
				"error at $.storage.trees.1: " + osStatName + " %FilesDir%" + string(filepath.Separator) + "nonexistent: " + osNotFound + "\n",
		},
		// Permissions and ownership
		{
			dirFiles: map[string]os.FileMode{
				"tree/file":        0600,
				"tree/subdir/file": 0644,
				"tree2/file":       0600,
			},
			dirLinks: map[string]string{
				"tree/subdir/link": "../file",
			},
			inTrees: []Tree{
				{
					Local:    "tree",
					FileMode: util.IntToPtr(0777),
					User: NodeUser{
						Name: util.StrToPtr("bovik"),
					},
					Group: NodeGroup{
						ID: util.IntToPtr(1000),
					},
				},
				{
					Local:   "tree2",
					DirMode: util.IntToPtr(0777),
					Path:    util.StrToPtr("/etc"),
				},
			},
			outDirs: []types.Directory{
				{
					Node: types.Node{
						Group: types.NodeGroup{
							ID: util.IntToPtr(1000),
						},
						Path: "/",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
					},
					DirectoryEmbedded1: types.DirectoryEmbedded1{
						Mode: util.IntToPtr(0755),
					},
				},
				{
					Node: types.Node{
						Group: types.NodeGroup{
							ID: util.IntToPtr(1000),
						},
						Path: "/subdir",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
					},
					DirectoryEmbedded1: types.DirectoryEmbedded1{
						Mode: util.IntToPtr(0755),
					},
				},
				{
					Node: types.Node{
						Path: "/etc",
					},
					DirectoryEmbedded1: types.DirectoryEmbedded1{
						Mode: util.IntToPtr(0777),
					},
				},
			},
			outFiles: []types.File{
				{
					Node: types.Node{
						Path: "/file",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
						Group: types.NodeGroup{
							ID: util.IntToPtr(1000),
						},
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0777),
					},
				},
				{
					Node: types.Node{
						Path: "/subdir/file",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
						Group: types.NodeGroup{
							ID: util.IntToPtr(1000),
						},
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree%2Fsubdir%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0777),
					},
				},
				{
					Node: types.Node{
						Path: "/etc/file",
					},
					FileEmbedded1: types.FileEmbedded1{
						Contents: types.Resource{
							Source:      util.StrToPtr("data:,tree2%2Ffile"),
							Compression: util.StrToPtr(""),
						},
						Mode: util.IntToPtr(0644),
					},
				},
			},
			outLinks: []types.Link{
				{
					Node: types.Node{
						Path: "/subdir/link",
						User: types.NodeUser{
							Name: util.StrToPtr("bovik"),
						},
						Group: types.NodeGroup{
							ID: util.IntToPtr(1000),
						},
					},
					LinkEmbedded1: types.LinkEmbedded1{
						Target: util.StrToPtr("../file"),
					},
				},
			},
		},
		// Overwrite via tree ownership fails
		{
			dirFiles: map[string]os.FileMode{
				"tree/etc/file": 0600,
			},
			inDirs: []Directory{
				{Path: "/etc"},
			},
			inTrees: []Tree{
				{
					Local:    "tree",
					FileMode: util.IntToPtr(0777),
					User: NodeUser{
						Name: util.StrToPtr("bovik"),
					},
					Group: NodeGroup{
						ID: util.IntToPtr(1000),
					},
				},
			},
			report: "error at $.storage.trees.0: " + common.ErrNodeExists.Error() + "\n",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			if test.skip != nil {
				// give the test an opportunity to skip
				test.skip(t)
			}
			filesDir := t.TempDir()
			for testPath, mode := range test.dirDirs {
				absPath := filepath.Join(filesDir, filepath.FromSlash(testPath))
				if err := os.MkdirAll(absPath, 0755); err != nil {
					t.Error(err)
					return
				}
				if err := os.Chmod(absPath, mode); err != nil {
					t.Error(err)
					return
				}
			}
			for testPath, mode := range test.dirFiles {
				absPath := filepath.Join(filesDir, filepath.FromSlash(testPath))
				if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
					t.Error(err)
					return
				}
				if err := os.WriteFile(absPath, []byte(testPath), mode); err != nil {
					t.Error(err)
					return
				}
			}
			for testPath, target := range test.dirLinks {
				absPath := filepath.Join(filesDir, filepath.FromSlash(testPath))
				if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
					t.Error(err)
					return
				}
				if err := os.Symlink(target, absPath); err != nil {
					t.Error(err)
					return
				}
			}
			for _, testPath := range test.dirSockets {
				absPath := filepath.Join(filesDir, filepath.FromSlash(testPath))
				if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
					t.Error(err)
					return
				}
				listener, err := net.ListenUnix("unix", &net.UnixAddr{
					Name: absPath,
					Net:  "unix",
				})
				if err != nil {
					t.Error(err)
					return
				}
				defer listener.Close()
			}

			config := Config{
				Storage: Storage{
					Files:       test.inFiles,
					Directories: test.inDirs,
					Links:       test.inLinks,
					Trees:       test.inTrees,
				},
			}
			options := common.TranslateOptions{
				FilesDir: filesDir,
			}
			if test.options != nil {
				options = *test.options
			}
			actual, translations, r := config.ToIgn3_7Unvalidated(options)

			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, config, r)
			expectedReport := strings.ReplaceAll(test.report, "%FilesDir%", filesDir)
			assert.Equal(t, expectedReport, r.String(), "bad report")
			if expectedReport != "" {
				return
			}
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")

			assert.Equal(t, test.outFiles, actual.Storage.Files, "files mismatch")
			assert.Equal(t, test.outDirs, actual.Storage.Directories, "directories mismatch")
			assert.Equal(t, test.outLinks, actual.Storage.Links, "links mismatch")
		})
	}
}

// TestTranslateIgnition tests translating the ct config.ignition to the ignition config.ignition section.
// It ensures that the version is set as well.
func TestTranslateIgnition(t *testing.T) {
	tests := []struct {
		in  Ignition
		out types.Ignition
	}{
		{
			Ignition{},
			types.Ignition{
				Version: "3.7.0-experimental",
			},
		},
		{
			Ignition{
				Config: IgnitionConfig{
					Merge: []Resource{
						{
							Inline: util.StrToPtr("xyzzy"),
						},
					},
					Replace: Resource{
						Inline: util.StrToPtr("xyzzy"),
					},
				},
			},
			types.Ignition{
				Version: "3.7.0-experimental",
				Config: types.IgnitionConfig{
					Merge: []types.Resource{
						{
							Source:      util.StrToPtr("data:,xyzzy"),
							Compression: util.StrToPtr(""),
						},
					},
					Replace: types.Resource{
						Source:      util.StrToPtr("data:,xyzzy"),
						Compression: util.StrToPtr(""),
					},
				},
			},
		},
		{
			Ignition{
				Proxy: Proxy{
					HTTPProxy: util.StrToPtr("https://example.com:8080"),
					NoProxy:   []string{"example.com"},
				},
			},
			types.Ignition{
				Version: "3.7.0-experimental",
				Proxy: types.Proxy{
					HTTPProxy: util.StrToPtr("https://example.com:8080"),
					NoProxy:   []types.NoProxyItem{types.NoProxyItem("example.com")},
				},
			},
		},
		{
			Ignition{
				Security: Security{
					TLS: TLS{
						CertificateAuthorities: []Resource{
							{
								Inline: util.StrToPtr("xyzzy"),
							},
						},
					},
				},
			},
			types.Ignition{
				Version: "3.7.0-experimental",
				Security: types.Security{
					TLS: types.TLS{
						CertificateAuthorities: []types.Resource{
							{
								Source:      util.StrToPtr("data:,xyzzy"),
								Compression: util.StrToPtr(""),
							},
						},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := translateIgnition(test.in, common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			// DebugVerifyCoverage wants to see a translation for $.version but
			// translateIgnition doesn't create one; ToIgn3_*Unvalidated handles
			// that since it has access to the Butane config version
			translations.AddTranslation(path.New("yaml", "bogus"), path.New("json", "version"))
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateKernelArguments tests translating the butane kernel_arguments.{should_exist,should_not_exist}.[i] entries to
// ignition kernelArguments.{shouldExist,shouldNotExist}.[i] entries.
//
// KernelArguments do not use a custom translation function (it utilizes the MergeP2 functionality) so pass an entire config
func TestTranslateKernelArguments(t *testing.T) {
	tests := []struct {
		in  Config
		out types.Config
	}{
		{
			Config{
				KernelArguments: KernelArguments{
					ShouldExist: []KernelArgument{
						"foo",
					},
					ShouldNotExist: []KernelArgument{
						"bar",
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				KernelArguments: types.KernelArguments{
					ShouldExist: []types.KernelArgument{
						"foo",
					},
					ShouldNotExist: []types.KernelArgument{
						"bar",
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToIgn3_7Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateLuks test translating the butane storage.luks.clevis.tang.[i] arguments to ignition storage.luks.clevis.tang.[i] entries.
func TestTranslateTang(t *testing.T) {
	tests := []struct {
		in  Config
		out types.Config
	}{
		// Luks with tang and all options set, returns a valid ignition config with the same options
		{
			Config{
				Storage: Storage{
					Filesystems: []Filesystem{
						{
							Device: "/dev/mapper/foo-bar",
							Path:   util.StrToPtr("/var/lib/containers"),
						},
					},
					Luks: []Luks{
						{
							Name:   "foo-bar",
							Device: util.StrToPtr("/dev/bar"),
							Clevis: Clevis{
								Tang: []Tang{
									{
										URL:           "http://example.com",
										Thumbprint:    util.StrToPtr("xyzzy"),
										Advertisement: util.StrToPtr("{\"payload\": \"xyzzy\"}"),
									},
								},
							},
						},
					},
				},
			},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Filesystems: []types.Filesystem{
						{
							Device: "/dev/mapper/foo-bar",
							Path:   util.StrToPtr("/var/lib/containers"),
						},
					},
					Luks: []types.Luks{
						{
							Name:   "foo-bar",
							Device: util.StrToPtr("/dev/bar"),
							Clevis: types.Clevis{
								Tang: []types.Tang{
									{
										URL:           "http://example.com",
										Thumbprint:    util.StrToPtr("xyzzy"),
										Advertisement: util.StrToPtr("{\"payload\": \"xyzzy\"}"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToIgn3_7Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateSSHAuthorizedKey tests translating the butane passwd.users[i].ssh_authorized_keys_local[j] entries to ignition passwd.users[i].ssh_authorized_keys[j] entries.
func TestTranslateSSHAuthorizedKey(t *testing.T) {
	sshKeyDir := t.TempDir()
	randomDir := t.TempDir()
	var sshKeyInline = "ssh-rsa AAAAAAAAA"
	var sshKey1 = "ssh-rsa BBBBBBBBB"
	var sshKey2 = "ssh-rsa CCCCCCCCC"
	var sshKey3 = "ssh-rsa DDDDDDDDD"
	var sshKeyFileName = "id_rsa.pub"
	var sshKeyMultipleKeysFileName = "multiple.pub"
	var sshKeyEmptyFileName = "empty.pub"
	var sshKeyBlankFileName = "blank.pub"
	var sshKeyWindowsLineEndingsFileName = "windows.pub"
	var sshKeyNonExistingFileName = "id_ed25519.pub"

	sshKeyData := map[string][]byte{
		sshKeyFileName:                   []byte(sshKey1),
		sshKeyMultipleKeysFileName:       []byte(fmt.Sprintf("%s\n#comment\n\n\n%s\n", sshKey2, sshKey3)),
		sshKeyEmptyFileName:              []byte(""),
		sshKeyBlankFileName:              []byte("\n\t"),
		sshKeyWindowsLineEndingsFileName: []byte(fmt.Sprintf("%s\r\n#comment\r\n", sshKey1)),
	}

	for fileName, contents := range sshKeyData {
		if err := os.WriteFile(filepath.Join(sshKeyDir, fileName), contents, 0644); err != nil {
			t.Error(err)
		}
	}

	tests := []struct {
		name         string
		in           PasswdUser
		out          types.PasswdUser
		translations []translate.Translation
		report       string
		fileDir      string
	}{
		{
			"empty user",
			PasswdUser{},
			types.PasswdUser{},
			[]translate.Translation{},
			"",
			sshKeyDir,
		},
		{
			"valid inline keys",
			PasswdUser{SSHAuthorizedKeys: []SSHAuthorizedKey{SSHAuthorizedKey(sshKeyInline)}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{types.SSHAuthorizedKey(sshKeyInline)}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid local keys",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyFileName}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{types.SSHAuthorizedKey(sshKey1)}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid local keys with multiple keys per file",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyMultipleKeysFileName}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{
				types.SSHAuthorizedKey(sshKey2),
				types.SSHAuthorizedKey("#comment"),
				types.SSHAuthorizedKey(sshKey3),
			}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 1)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 2)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid multiple local key files",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyFileName, sshKeyMultipleKeysFileName}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{
				types.SSHAuthorizedKey(sshKey1),
				types.SSHAuthorizedKey(sshKey2),
				types.SSHAuthorizedKey("#comment"),
				types.SSHAuthorizedKey(sshKey3),
			}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 1), To: path.New("json", "sshAuthorizedKeys", 1)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 1), To: path.New("json", "sshAuthorizedKeys", 2)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 1), To: path.New("json", "sshAuthorizedKeys", 3)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid local and inline keys",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyFileName}, SSHAuthorizedKeys: []SSHAuthorizedKey{SSHAuthorizedKey(sshKeyInline)}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{types.SSHAuthorizedKey(sshKeyInline), types.SSHAuthorizedKey(sshKey1)}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 1)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid local keys with multiple keys per file and inline keys",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyMultipleKeysFileName}, SSHAuthorizedKeys: []SSHAuthorizedKey{SSHAuthorizedKey(sshKeyInline)}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{
				types.SSHAuthorizedKey(sshKeyInline),
				types.SSHAuthorizedKey(sshKey2),
				types.SSHAuthorizedKey("#comment"),
				types.SSHAuthorizedKey(sshKey3),
			}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 1)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 2)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 3)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid empty local file",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyEmptyFileName}},
			types.PasswdUser{},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
			},
			"",
			sshKeyDir,
		},
		{
			"valid blank local file",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyBlankFileName}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{types.SSHAuthorizedKey("\t")}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
			},
			"",
			sshKeyDir,
		},
		{
			"valid Windows style line endings in local file",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyWindowsLineEndingsFileName}},
			types.PasswdUser{SSHAuthorizedKeys: []types.SSHAuthorizedKey{
				types.SSHAuthorizedKey(sshKey1),
				types.SSHAuthorizedKey("#comment"),
			}},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 0)},
				{From: path.New("yaml", "ssh_authorized_keys_local", 0), To: path.New("json", "sshAuthorizedKeys", 1)},
			},
			"",
			sshKeyDir,
		},
		{
			"missing local file",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyNonExistingFileName}},
			types.PasswdUser{},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
			},
			"error at $.ssh_authorized_keys_local.0: open " + filepath.Join(sshKeyDir, sshKeyNonExistingFileName) + ": " + osNotFound + "\n",
			sshKeyDir,
		},
		{
			"missing embed directory",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyFileName}},
			types.PasswdUser{},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
			},
			"error at $.ssh_authorized_keys_local: " + common.ErrNoFilesDir.Error() + "\n",
			"",
		},
		{
			"wrong embed directory",
			PasswdUser{SSHAuthorizedKeysLocal: []string{sshKeyFileName}},
			types.PasswdUser{},
			[]translate.Translation{
				{From: path.New("yaml", "ssh_authorized_keys_local"), To: path.New("json", "sshAuthorizedKeys")},
			},
			"error at $.ssh_authorized_keys_local.0: open " + filepath.Join(randomDir, sshKeyFileName) + ": " + osNotFound + "\n",
			randomDir,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, translations, r := translatePasswdUser(test.in, common.TranslateOptions{FilesDir: test.fileDir})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, test.report, r.String(), "bad report")
			baseutil.VerifyTranslations(t, translations, test.translations)
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestTranslateUnitLocal tests translating the butane systemd.units[i].contents_local entries to ignition systemd.units[i].contents entries.
func TestTranslateUnitLocal(t *testing.T) {
	unitDir := t.TempDir()
	randomDir := t.TempDir()
	var unitName = "example.service"
	var dropinName = "example.conf"
	var unitDefinitionInline = "[Service]\nExecStart=/bin/false\n"
	var unitDefinitionFile = "[Service]\nExecStart=/bin/true\n"
	var unitEmptyFileName = "empty.service"
	var unitEmptyDefinition = ""
	var unitNonExistingFileName = "random.service"

	err := os.WriteFile(filepath.Join(unitDir, unitName), []byte(unitDefinitionFile), 0644)
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(filepath.Join(unitDir, unitEmptyFileName), []byte(unitEmptyDefinition), 0644)
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name         string
		in           Unit
		out          types.Unit
		translations []translate.Translation
		report       string
		fileDir      string
	}{
		{
			"empty unit",
			Unit{},
			types.Unit{},
			[]translate.Translation{},
			"",
			"",
		},
		{
			"valid contents",
			Unit{Contents: &unitDefinitionInline, Name: unitName},
			types.Unit{Contents: &unitDefinitionInline, Name: unitName},
			[]translate.Translation{},
			"",
			"",
		},
		{
			"valid contents_local",
			Unit{ContentsLocal: &unitName, Name: unitName},
			types.Unit{Contents: &unitDefinitionFile, Name: unitName},
			[]translate.Translation{
				{From: path.New("yaml", "contents_local"), To: path.New("json", "contents")},
			},
			"",
			unitDir,
		},
		{
			"non existing contents_local file name",
			Unit{ContentsLocal: &unitNonExistingFileName, Name: unitName},
			types.Unit{Name: unitName},
			[]translate.Translation{},
			"error at $.contents_local: open " + filepath.Join(unitDir, unitNonExistingFileName) + ": " + osNotFound + "\n",
			unitDir,
		},
		{
			"valid empty contents_local file",
			Unit{ContentsLocal: &unitEmptyFileName, Name: unitName},
			types.Unit{Contents: &unitEmptyDefinition, Name: unitName},
			[]translate.Translation{
				{From: path.New("yaml", "contents_local"), To: path.New("json", "contents")},
			},
			"",
			unitDir,
		},
		{
			"missing embed directory",
			Unit{ContentsLocal: &unitName, Name: unitName},
			types.Unit{Name: unitName},
			[]translate.Translation{},
			"error at $.contents_local: " + common.ErrNoFilesDir.Error() + "\n",
			"",
		},
		{
			"wrong embed directory",
			Unit{ContentsLocal: &unitName, Name: unitName},
			types.Unit{Name: unitName},
			[]translate.Translation{},
			"error at $.contents_local: open " + filepath.Join(randomDir, unitName) + ": " + osNotFound + "\n",
			randomDir,
		},
		{
			"empty dropin unit",
			Unit{Name: dropinName, Dropins: nil},
			types.Unit{Name: dropinName, Dropins: nil},
			[]translate.Translation{},
			"",
			"",
		},
		{
			"valid dropin contents",
			Unit{Dropins: []Dropin{{Name: dropinName, Contents: &unitDefinitionInline}}, Name: unitName},
			types.Unit{Dropins: []types.Dropin{{Name: dropinName, Contents: &unitDefinitionInline}}, Name: unitName},
			[]translate.Translation{},
			"",
			"",
		},
		{
			"valid dropin contents_local",
			Unit{Dropins: []Dropin{{Name: dropinName, ContentsLocal: &unitName}}, Name: unitName},
			types.Unit{Dropins: []types.Dropin{{Name: dropinName, Contents: &unitDefinitionFile}}, Name: unitName},
			[]translate.Translation{
				{From: path.New("yaml", "dropins", 0, "contents_local"), To: path.New("json", "dropins", 0, "contents")},
			},
			"",
			unitDir,
		},
		{
			"non existing dropin contents_local file name",
			Unit{Dropins: []Dropin{{Name: dropinName, ContentsLocal: &unitNonExistingFileName}}, Name: unitName},
			types.Unit{Dropins: []types.Dropin{{Name: dropinName}}, Name: unitName},
			[]translate.Translation{},
			"error at $.dropins.0.contents_local: open " + filepath.Join(unitDir, unitNonExistingFileName) + ": " + osNotFound + "\n",
			unitDir,
		},
		{
			"valid empty dropin contents_local file",
			Unit{Dropins: []Dropin{{Name: dropinName, ContentsLocal: &unitEmptyFileName}}, Name: unitName},
			types.Unit{Dropins: []types.Dropin{{Name: dropinName, Contents: &unitEmptyDefinition}}, Name: unitName},
			[]translate.Translation{
				{From: path.New("yaml", "dropins", 0, "contents_local"), To: path.New("json", "dropins", 0, "contents")},
			},
			"",
			unitDir,
		},
		{
			"missing embed directory for dropin",
			Unit{Dropins: []Dropin{{Name: dropinName, ContentsLocal: &unitName}}, Name: unitName},
			types.Unit{Dropins: []types.Dropin{{Name: dropinName}}, Name: unitName},
			[]translate.Translation{},
			"error at $.dropins.0.contents_local: " + common.ErrNoFilesDir.Error() + "\n",
			"",
		},
		{
			"wrong embed directory for dropin",
			Unit{Dropins: []Dropin{{Name: dropinName, ContentsLocal: &unitName}}, Name: unitName},
			types.Unit{Dropins: []types.Dropin{{Name: dropinName}}, Name: unitName},
			[]translate.Translation{},
			"error at $.dropins.0.contents_local: open " + filepath.Join(randomDir, unitName) + ": " + osNotFound + "\n",
			randomDir,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, translations, r := translateUnit(test.in, common.TranslateOptions{FilesDir: test.fileDir})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, test.report, r.String(), "bad report")
			baseutil.VerifyTranslations(t, translations, test.translations)
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

// TestToIgn3_7 tests the config.ToIgn3_7 function ensuring it will generate a valid config even when empty. Not much else is
// tested since it uses the Ignition translation code which has its own set of tests.
func TestToIgn3_7(t *testing.T) {
	tests := []struct {
		in  Config
		out types.Config
	}{
		{
			Config{},
			types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
			},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			actual, translations, r := test.in.ToIgn3_7Unvalidated(common.TranslateOptions{})
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, test.out, actual, "translation mismatch")
			assert.Equal(t, report.Report{}, r, "non-empty report")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}

func TestTranslateQuadlets(t *testing.T) {
	const (
		SleepContainer = `[Unit]
Description=A sleepy container
[Container]
ContainerName=sleepy-pod-inf
Image=quay.io/fedora/fedora
Exec=sleep infinity
[Install]
WantedBy=multi-user.target`

		SleepContainerTemplate = `[Unit]
Description=A templated sleepy container
[Container]
Image=quay.io/fedora/fedora
Exec=sleep %i
[Service]
# Restart service when sleep finishes
Restart=always
[Install]
WantedBy=multi-user.target`
	)

	sleepContainerAsData, sleepContainerCompression := baseutil.CompressDataURL(t, []byte(SleepContainer))
	sleepContainerTemplateAsData, sleepContainerTemplateCompression := baseutil.CompressDataURL(t, []byte(SleepContainerTemplate))

	filesDir := t.TempDir()
	fileContents := map[string]string{
		"sample.container":  SleepContainer,
		"sample@.container": SleepContainerTemplate,
		"foo.conf":          "[Service]\nTimeoutStartSec=900",
	}
	for name, contents := range fileContents {
		if err := os.MkdirAll(filepath.Join(filesDir, filepath.Dir(name)), 0755); err != nil {
			t.Error(err)
			return
		}
		err := os.WriteFile(filepath.Join(filesDir, name), []byte(contents), 0644)
		if err != nil {
			t.Error(err)
			return
		}
	}

	tests := []struct {
		name        string
		inputConfig Config
		outConf     types.Config
		reportPath  string
		options     common.TranslateOptions
	}{
		{
			name: "Basic .container quadlets",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  true,
						},
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  false,
						},
					},
				},
			},
			outConf: types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy.container",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr(sleepContainerAsData),
									Compression: util.StrToPtr(sleepContainerCompression),
								},
								Mode: util.IntToPtr(0644),
							},
						},
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/users/sleepy.container",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr(sleepContainerAsData),
									Compression: util.StrToPtr(sleepContainerCompression),
								},
								Mode: util.IntToPtr(0644),
							},
						},
					},
				},
			},
		},
		{
			name: "Template instance is symlink",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:     "sleepy@.container",
							Contents: util.StrToPtr(SleepContainerTemplate),
							Rootful:  true,
						},
						{
							Name:    "sleepy@100.container",
							Rootful: true,
						},
					},
				},
			},
			outConf: types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy@.container",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr(sleepContainerTemplateAsData),
									Compression: util.StrToPtr(sleepContainerTemplateCompression),
								},
								Mode: util.IntToPtr(0644),
							},
						},
					},
					Links: []types.Link{
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy@100.container",
							},
							LinkEmbedded1: types.LinkEmbedded1{
								Target: util.StrToPtr("sleepy@.container"),
							},
						},
					},
				},
			},
		},
		{
			name: "Quadlet with non-existent contents_local",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:          "sleepy.container",
							ContentsLocal: util.StrToPtr("fake-file.container"),
							Rootful:       true,
						},
					},
				},
			},
			options:    common.TranslateOptions{FilesDir: filesDir},
			reportPath: "error at $.systemd.quadlets.0.contents_local: open " + filepath.Join(filesDir, "fake-file.container") + ": " + osNotFound + "\n",
		},
		{
			name: "Quadlet with contents_local",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:          "sleepy.container",
							ContentsLocal: util.StrToPtr("sample.container"),
							Rootful:       true,
						},
					},
				},
			},
			outConf: types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy.container",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr(sleepContainerAsData),
									Compression: util.StrToPtr(sleepContainerCompression),
								},
								Mode: util.IntToPtr(0644),
							},
						},
					},
				},
			},
			options: common.TranslateOptions{FilesDir: filesDir},
		},
		{
			name: "Overrides break",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						// two quadlets with the same name should give an error
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  true,
						},
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainerTemplate),
							Rootful:  true,
						},
					},
				},
			},
			outConf:    types.Config{},
			reportPath: "error at $.systemd.quadlets.1: matching filesystem node has existing contents or different type\n",
		},
		{
			name: "Template with dropin",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:     "sleepy@.container",
							Contents: util.StrToPtr(SleepContainerTemplate),
							Rootful:  true,
							Dropins: []Dropin{
								{
									Name:     "sample.conf",
									Contents: util.StrToPtr("[Service]\nTimeoutStartSec=900"),
								},
							},
						},
						{
							Name:    "sleepy@100.container",
							Rootful: true,
							Dropins: []Dropin{
								{
									Name:          "foo.conf",
									ContentsLocal: util.StrToPtr("foo.conf"),
								},
							},
						},
					},
				},
			},
			outConf: types.Config{
				Ignition: types.Ignition{
					Version: "3.7.0-experimental",
				},
				Storage: types.Storage{
					Files: []types.File{
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy@.container",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr(sleepContainerTemplateAsData),
									Compression: util.StrToPtr(sleepContainerTemplateCompression),
								},
								Mode: util.IntToPtr(0644),
							},
						},
						{ // Dropin for all sleepy@.container instances
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy@.container.d/sample.conf",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr("data:,%5BService%5D%0ATimeoutStartSec%3D900"),
									Compression: util.StrToPtr(""),
								},
								Mode: util.IntToPtr(0644),
							},
						},
						{ // Dropin for the instance of sleepy@.container with 100 as input
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy@100.container.d/foo.conf",
							},
							FileEmbedded1: types.FileEmbedded1{
								Contents: types.Resource{
									Source:      util.StrToPtr("data:,%5BService%5D%0ATimeoutStartSec%3D900"),
									Compression: util.StrToPtr(""),
								},
								Mode: util.IntToPtr(0644),
							},
						},
					},
					Links: []types.Link{
						{
							Node: types.Node{
								Path: "/etc/containers/systemd/sleepy@100.container",
							},
							LinkEmbedded1: types.LinkEmbedded1{
								Target: util.StrToPtr("sleepy@.container"),
							},
						},
					},
				},
			},
			options: common.TranslateOptions{FilesDir: filesDir},
		},
		{
			name: "Duplicate names will lead to file-collision",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  true,
						},
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  true,
						},
					},
				},
			},
			reportPath: "error at $.systemd.quadlets.1: matching filesystem node has existing contents or different type\n",
		},
		{
			name: "File collision with trees",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:     "sample.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  true,
						},
					},
				},
				Storage: Storage{
					Trees: []Tree{
						{
							Path: util.StrToPtr("/etc/containers/systemd"),
						},
					},
				},
			},
			options:    common.TranslateOptions{FilesDir: filesDir},
			reportPath: "error at $.systemd.quadlets.0: matching filesystem node has existing contents or different type\n",
		},
		{
			name: "File collision",
			inputConfig: Config{
				Systemd: Systemd{
					Quadlets: []Quadlet{
						{
							Name:     "sleepy.container",
							Contents: util.StrToPtr(SleepContainer),
							Rootful:  true,
						},
					},
				},
				Storage: Storage{
					Files: []File{
						{
							Path: "/etc/containers/systemd/sleepy.container",
						},
					},
				},
			},
			reportPath: "error at $.systemd.quadlets.0: matching filesystem node has existing contents or different type\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _, r := test.inputConfig.ToIgn3_7Unvalidated(test.options)
			assert.Equal(t, test.outConf, c, "translation mismatch")
			assert.Equal(t, test.reportPath, r.String(), "report mismatch")
		})
	}
}
