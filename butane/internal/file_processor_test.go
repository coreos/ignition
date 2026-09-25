// Copyright 2026 Red Hat, Inc
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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/butane/config"
	"github.com/coreos/ignition/v2/butane/config/common"
)

func TestProcessFile(t *testing.T) {
	processor := filepath.Join(t.TempDir(), "processor")
	if err := os.WriteFile(processor, []byte("#!/bin/sh\ntr '[:lower:]' '[:upper:]'\n"), 0755); err != nil {
		t.Fatal(err)
	}

	output, err := processFile(strings.NewReader("hello world"), processor)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "HELLO WORLD" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestLocalFileReaderIsPerTranslation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "message.txt"), []byte("from disk"), 0644); err != nil {
		t.Fatal(err)
	}
	input := []byte("variant: fcos\nversion: 1.7.0\nstorage:\n  files:\n    - path: /etc/message\n      contents:\n        local: message.txt\n")
	for _, test := range []struct {
		name   string
		reader func(string) ([]byte, error)
		want   string
	}{
		{name: "default", want: "data:,from%20disk"},
		{name: "first", reader: func(string) ([]byte, error) { return []byte("first"), nil }, want: "data:,first"},
		{name: "second", reader: func(string) ([]byte, error) { return []byte("second"), nil }, want: "data:,second"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output, _, err := config.TranslateBytes(input, common.TranslateBytesOptions{
				TranslateOptions: common.TranslateOptions{FilesDir: dir, LocalFileReader: test.reader},
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(output), test.want) {
				t.Fatalf("expected %q in %s", test.want, output)
			}
		})
	}
}
