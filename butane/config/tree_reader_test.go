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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/butane/config"
	"github.com/coreos/ignition/v2/butane/config/common"
)

func TestTreeReaderProcessesRegularFilePayload(t *testing.T) {
	filesDir := t.TempDir()
	payloadPath := filepath.Join(filesDir, "tree", "nested", "payload.bin")
	writeReaderTestFile(t, payloadPath, []byte{0x01, 0x00, 0xfe, 0xff})
	want := []byte{0x00, 0x02, 0xfd, 0x00}
	var calls int
	reader := func(path string) ([]byte, error) {
		if filepath.Clean(path) != payloadPath {
			t.Errorf("reader path = %q, want %q", path, payloadPath)
		}
		calls++
		return want, nil
	}

	output := translateReaderTestConfig(t, "1.7.0", `storage:
  trees:
    - local: tree
`, common.TranslateOptions{FilesDir: filesDir, LocalFileReader: reader})
	if got := decodeReaderTestFile(t, output, "/nested/payload.bin", nil); !bytes.Equal(got, want) {
		t.Fatalf("embedded binary contents = %v, want %v", got, want)
	}
	if calls != 1 {
		t.Fatalf("reader called %d times, want 1", calls)
	}
}

func TestTreeReaderDefaultsToDiskContents(t *testing.T) {
	filesDir := t.TempDir()
	want := []byte{0xff, 0x00, 0x07, 0x80}
	writeReaderTestFile(t, filepath.Join(filesDir, "tree", "payload.bin"), want)

	output := translateReaderTestConfig(t, "1.7.0", `storage:
  trees:
    - local: tree
`, common.TranslateOptions{FilesDir: filesDir})
	if got := decodeReaderTestFile(t, output, "/payload.bin", nil); !bytes.Equal(got, want) {
		t.Fatalf("embedded binary contents = %v, want %v", got, want)
	}
}

func TestTreeReaderContinuesAfterReadErrors(t *testing.T) {
	filesDir := t.TempDir()
	writeReaderTestFile(t, filepath.Join(filesDir, "tree", "one.txt"), []byte("one"))
	writeReaderTestFile(t, filepath.Join(filesDir, "tree", "two.txt"), []byte("two"))
	input := []byte("variant: fcos\nversion: 1.7.0\nstorage:\n  trees:\n    - local: tree\n")
	_, report, err := config.TranslateBytes(input, common.TranslateBytesOptions{
		Raw: true,
		TranslateOptions: common.TranslateOptions{
			FilesDir: filesDir,
			LocalFileReader: func(string) ([]byte, error) {
				return nil, errors.New("tree reader sentinel")
			},
		},
	})
	if err == nil {
		t.Fatal("expected tree reader errors")
	}
	count := 0
	for _, entry := range report.Entries {
		if strings.Contains(entry.Message, "tree reader sentinel") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("got %d tree read errors, want 2; report: %v", count, report.Entries)
	}
}

func TestTreeReaderValidatesCompressionAfterReading(t *testing.T) {
	filesDir := t.TempDir()
	writeReaderTestFile(t, filepath.Join(filesDir, "tree", "payload.txt"), []byte("on-disk bytes are ignored"))
	input := `storage:
  files:
    - path: /payload.txt
      contents:
        compression: gzip
  trees:
    - local: tree
`
	validGzip := gzipReaderTestContents(t, []byte("processed tree contents"))
	output := translateReaderTestConfig(t, "1.7.0", input, common.TranslateOptions{
		FilesDir: filesDir,
		LocalFileReader: func(string) ([]byte, error) {
			return validGzip, nil
		},
	})
	if got := decodeReaderTestFile(t, output, "/payload.txt", stringPtrReaderTest("gzip")); string(got) != "processed tree contents" {
		t.Fatalf("decoded tree contents = %q", got)
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
		t.Fatal("expected invalid processor output for gzip tree resource to fail")
	}
	if !readerTestReportContains(report.Entries, common.ErrCompressionInvalid.Error()) {
		t.Fatalf("report %v does not contain %q", report.Entries, common.ErrCompressionInvalid)
	}
}

func TestTreeReaderDoesNotChangeTreeEnumeration(t *testing.T) {
	filesDir := t.TempDir()
	writeReaderTestFile(t, filepath.Join(filesDir, "tree", "payload.bin"), []byte{0x11, 0x22})
	var seen []string
	output := translateReaderTestConfig(t, "1.7.0", `storage:
  trees:
    - local: tree
`, common.TranslateOptions{
		FilesDir: filesDir,
		LocalFileReader: func(path string) ([]byte, error) {
			seen = append(seen, path)
			return os.ReadFile(path)
		},
	})
	if len(seen) != 1 || filepath.Base(seen[0]) != "payload.bin" {
		t.Fatalf("reader calls = %v, want only payload.bin", seen)
	}
	if got := decodeReaderTestFile(t, output, "/payload.bin", nil); !bytes.Equal(got, []byte{0x11, 0x22}) {
		t.Fatalf("embedded contents = %v", got)
	}
}
