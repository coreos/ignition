// Copyright 2026 Red Hat, Inc.
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

package config_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coreos/ignition/v2/butane/base/util"
	"github.com/coreos/ignition/v2/butane/config"
	"github.com/coreos/ignition/v2/butane/config/common"
	"github.com/coreos/vcontext/report"
	"github.com/vincent-petithory/dataurl"
)

type readerTestConfig struct {
	Storage struct {
		Files []struct {
			Path     string `json:"path"`
			Contents struct {
				Source      string  `json:"source"`
				Compression *string `json:"compression"`
			} `json:"contents"`
		} `json:"files"`
	} `json:"storage"`
}

func TestLocalFileReaderAcrossFCOSBasePackages(t *testing.T) {
	for _, version := range []string{
		"1.1.0",              // base/v0_2
		"1.2.0",              // base/v0_3 (also used by 1.3.0)
		"1.4.0",              // base/v0_4
		"1.5.0",              // base/v0_5
		"1.6.0",              // base/v0_6
		"1.7.0",              // base/v0_7
		"1.8.0-experimental", // base/v0_8_exp
	} {
		t.Run(version, func(t *testing.T) {
			filesDir := t.TempDir()
			writeReaderTestFile(t, filepath.Join(filesDir, "payload.txt"), []byte("disk contents"))
			want := "processed by reader for " + version
			var calls atomic.Int32
			reader := func(path string) ([]byte, error) {
				if filepath.Clean(path) != filepath.Join(filesDir, "payload.txt") {
					t.Errorf("reader path = %q, want a file inside %q", path, filesDir)
				}
				if err := util.EnsurePathWithinFilesDir(path, filesDir); err != nil {
					t.Errorf("reader path %q escaped files directory %q: %v", path, filesDir, err)
				}
				calls.Add(1)
				return []byte(want), nil
			}

			output := translateReaderTestConfig(t, version, `storage:
  files:
    - path: /etc/reader-test
      contents:
        local: payload.txt
`, common.TranslateOptions{FilesDir: filesDir, LocalFileReader: reader})
			if got := decodeReaderTestFile(t, output, "/etc/reader-test", nil); string(got) != want {
				t.Fatalf("embedded contents = %q, want %q", got, want)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("reader called %d times, want 1", got)
			}
		})
	}
}

func TestNilLocalFileReaderUsesDiskContents(t *testing.T) {
	filesDir := t.TempDir()
	want := []byte("ordinary local file contents")
	writeReaderTestFile(t, filepath.Join(filesDir, "payload.txt"), want)

	output := translateReaderTestConfig(t, "1.7.0", `storage:
  files:
    - path: /etc/reader-test
      contents:
        local: payload.txt
`, common.TranslateOptions{FilesDir: filesDir})
	if got := decodeReaderTestFile(t, output, "/etc/reader-test", nil); !bytes.Equal(got, want) {
		t.Fatalf("embedded contents = %q, want %q", got, want)
	}
}

func TestLocalFileReaderIsPerTranslation(t *testing.T) {
	filesDir := t.TempDir()
	writeReaderTestFile(t, filepath.Join(filesDir, "payload.txt"), []byte("disk contents"))
	input := `storage:
  files:
    - path: /etc/reader-test
      contents:
        local: payload.txt
`

	for _, value := range []string{"reader one", "reader two"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			output := translateReaderTestConfig(t, "1.7.0", input, common.TranslateOptions{
				FilesDir: filesDir,
				LocalFileReader: func(string) ([]byte, error) {
					return []byte(value), nil
				},
			})
			if got := decodeReaderTestFile(t, output, "/etc/reader-test", nil); string(got) != value {
				t.Fatalf("embedded contents = %q, want %q", got, value)
			}
		})
	}
}

func TestLocalFileReaderErrorsStayWithinFilesDir(t *testing.T) {
	filesDir := t.TempDir()
	writeReaderTestFile(t, filepath.Join(filepath.Dir(filesDir), "outside-reader-test.txt"), []byte("outside"))
	called := false
	input := `storage:
  files:
    - path: /etc/reader-test
      contents:
        local: ../outside-reader-test.txt
`
	_, report, err := config.TranslateBytes([]byte("variant: fcos\nversion: 1.7.0\n"+input), common.TranslateBytesOptions{
		Raw: true,
		TranslateOptions: common.TranslateOptions{
			FilesDir: filesDir,
			LocalFileReader: func(string) ([]byte, error) {
				called = true
				return []byte("must not be read"), nil
			},
		},
	})
	if err == nil {
		t.Fatal("expected path traversal to fail")
	}
	if called {
		t.Fatal("reader was called for a path outside FilesDir")
	}
	if !readerTestReportContains(report.Entries, common.ErrFilesDirEscape.Error()) {
		t.Fatalf("report %v does not contain %q", report.Entries, common.ErrFilesDirEscape)
	}
}

func TestLocalFileReaderValidatesCompressionAfterReading(t *testing.T) {
	filesDir := t.TempDir()
	writeReaderTestFile(t, filepath.Join(filesDir, "payload.gz"), []byte("on-disk bytes are ignored"))
	validGzip := gzipReaderTestContents(t, []byte("processed and compressed"))
	input := `storage:
  files:
    - path: /etc/reader-test
      contents:
        compression: gzip
        local: payload.gz
`

	output := translateReaderTestConfig(t, "1.7.0", input, common.TranslateOptions{
		FilesDir: filesDir,
		LocalFileReader: func(string) ([]byte, error) {
			return validGzip, nil
		},
	})
	if got := decodeReaderTestFile(t, output, "/etc/reader-test", stringPtrReaderTest("gzip")); string(got) != "processed and compressed" {
		t.Fatalf("decoded contents = %q", got)
	}

	_, report, err := config.TranslateBytes([]byte("variant: fcos\nversion: 1.7.0\n"+input), common.TranslateBytesOptions{
		Raw: true,
		TranslateOptions: common.TranslateOptions{
			FilesDir: filesDir,
			LocalFileReader: func(string) ([]byte, error) {
				return []byte("not gzip"), nil
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid processor output for gzip resource to fail")
	}
	if !readerTestReportContains(report.Entries, common.ErrCompressionInvalid.Error()) {
		t.Fatalf("report %v does not contain %q", report.Entries, common.ErrCompressionInvalid)
	}
}

func translateReaderTestConfig(t *testing.T, version, body string, options common.TranslateOptions) []byte {
	t.Helper()
	input := []byte(fmt.Sprintf("variant: fcos\nversion: %s\n%s", version, body))
	output, report, err := config.TranslateBytes(input, common.TranslateBytesOptions{
		Raw:              true,
		TranslateOptions: options,
	})
	if err != nil {
		t.Fatalf("translate failed: %v (report: %v)", err, report.Entries)
	}
	if len(report.Entries) != 0 {
		t.Fatalf("unexpected translation report: %v", report.Entries)
	}
	return output
}

func decodeReaderTestFile(t *testing.T, output []byte, path string, compression *string) []byte {
	t.Helper()
	var cfg readerTestConfig
	if err := json.Unmarshal(output, &cfg); err != nil {
		t.Fatalf("unmarshal Ignition output: %v\n%s", err, output)
	}
	for _, file := range cfg.Storage.Files {
		if file.Path != path {
			continue
		}
		if compression != nil && (file.Contents.Compression == nil || *file.Contents.Compression != *compression) {
			t.Fatalf("compression = %v, want %q", file.Contents.Compression, *compression)
		}
		data, err := dataurl.DecodeString(file.Contents.Source)
		if err != nil {
			t.Fatalf("decode data URL: %v", err)
		}
		if file.Contents.Compression != nil && *file.Contents.Compression == "gzip" {
			reader, err := gzip.NewReader(bytes.NewReader(data.Data))
			if err != nil {
				t.Fatalf("open gzip contents: %v", err)
			}
			decompressed, err := io.ReadAll(reader)
			closeErr := reader.Close()
			if err != nil {
				t.Fatalf("decompress contents: %v", err)
			}
			if closeErr != nil {
				t.Fatalf("close gzip reader: %v", closeErr)
			}
			return decompressed
		}
		return data.Data
	}
	t.Fatalf("file %q not found in Ignition output: %s", path, output)
	return nil
}

func gzipReaderTestContents(t *testing.T, input []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(input); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeReaderTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
}

func readerTestReportContains(entries []report.Entry, want string) bool {
	for _, entry := range entries {
		if strings.Contains(entry.Message, want) {
			return true
		}
	}
	return false
}

func stringPtrReaderTest(s string) *string { return &s }
