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
