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

package main

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProcessFileWithoutProcessorReadsRawInput(t *testing.T) {
	const input = "raw input\n"
	got, err := processFile(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("processFile returned an error: %v", err)
	}
	if string(got) != input {
		t.Fatalf("processFile returned %q, want %q", got, input)
	}
}

func TestProcessFileRunsProcessorDirectlyWithPathContainingSpaces(t *testing.T) {
	processor := writePOSIXProcessor(t, "processor with spaces", "tr '[:lower:]' '[:upper:]'\n")
	got, err := processFile(strings.NewReader("hello from processor\n"), processor)
	if err != nil {
		t.Fatalf("processFile returned an error: %v", err)
	}
	if string(got) != "HELLO FROM PROCESSOR\n" {
		t.Fatalf("processFile returned %q, want uppercase output", got)
	}
}

func TestProcessFilePreservesExitError(t *testing.T) {
	processor := writePOSIXProcessor(t, "failing processor", "cat\nprintf 'processor output'\nexit 7\n")
	got, err := processFile(strings.NewReader("input"), processor)
	if err == nil {
		t.Fatal("processFile succeeded, want processor failure")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("processFile error %T does not preserve *exec.ExitError: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("processor exit code = %d, want 7", exitErr.ExitCode())
	}
	if got != nil {
		t.Fatalf("processFile returned output %q on failure, want nil", got)
	}
}

func TestProcessFileMissingExecutable(t *testing.T) {
	processor := filepath.Join(t.TempDir(), "missing processor")
	_, err := processFile(strings.NewReader("input"), processor)
	if err == nil {
		t.Fatal("processFile succeeded, want missing-executable error")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		t.Fatalf("missing executable unexpectedly returned *exec.ExitError: %v", err)
	}
	if !strings.Contains(err.Error(), processor) {
		t.Fatalf("error %q does not identify processor %q", err, processor)
	}
}

func TestProcessLocalFile(t *testing.T) {
	processor := writePOSIXProcessor(t, "local processor", "tr '[:lower:]' '[:upper:]'\n")
	inputPath := filepath.Join(t.TempDir(), "input with spaces.txt")
	if err := os.WriteFile(inputPath, []byte("local input\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := processLocalFile(inputPath, processor)
	if err != nil {
		t.Fatalf("processLocalFile returned an error: %v", err)
	}
	if string(got) != "LOCAL INPUT\n" {
		t.Fatalf("processLocalFile returned %q, want uppercase output", got)
	}
}

func TestProcessLocalFileMissingInput(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "missing input")
	_, err := processLocalFile(inputPath, "")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("processLocalFile error = %v, want fs.ErrNotExist", err)
	}
}

func writePOSIXProcessor(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	path := filepath.Join(t.TempDir(), name)
	contents := "#!/bin/sh\n" + body
	if err := os.WriteFile(path, []byte(contents), 0700); err != nil {
		t.Fatalf("writing processor fixture: %v", err)
	}
	return path
}
