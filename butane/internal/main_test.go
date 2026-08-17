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
// limitations under the License.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadInput(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (input string, cleanup func())
		wantData []byte
		wantErr  bool
	}{
		{
			name: "stdin",
			setup: func(t *testing.T) (string, func()) {
				content := []byte("hello from stdin")
				orig := os.Stdin
				tmp, err := os.CreateTemp("", "butane-stdin")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				if _, err := tmp.Write(content); err != nil {
					t.Fatalf("failed to write temp file: %v", err)
				}
				if _, err := tmp.Seek(0, 0); err != nil {
					t.Fatalf("failed to seek temp file: %v", err)
				}
				os.Stdin = tmp
				return "", func() {
					os.Stdin = orig
					tmp.Close()
					os.Remove(tmp.Name())
				}
			},
			wantData: []byte("hello from stdin"),
			wantErr:  false,
		},
		{
			name: "empty stdin",
			setup: func(t *testing.T) (string, func()) {
				orig := os.Stdin
				tmp, err := os.CreateTemp("", "butane-stdin-empty")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				if _, err := tmp.Seek(0, 0); err != nil {
					t.Fatalf("failed to seek temp file: %v", err)
				}
				os.Stdin = tmp
				return "", func() {
					os.Stdin = orig
					tmp.Close()
					os.Remove(tmp.Name())
				}
			},
			wantData: []byte{},
			wantErr:  false,
		},
		{
			name: "file",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				path := filepath.Join(dir, "input.bu")
				content := []byte("variant: fcos")
				if err := os.WriteFile(path, content, 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path, func() {}
			},
			wantData: []byte("variant: fcos"),
			wantErr:  false,
		},
		{
			name: "missing file",
			setup: func(t *testing.T) (string, func()) {
				missing := filepath.Join(t.TempDir(), "does-not-exist")
				return missing, func() {}
			},
			wantData: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, cleanup := tt.setup(t)
			defer cleanup()

			data, filename, err := readInput(input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(data) != string(tt.wantData) {
				t.Errorf("expected data %q, got %q", tt.wantData, data)
			}

			wantName := "<stdin>"
			if input != "" {
				wantName = input
			}
			if filename != wantName {
				t.Errorf("expected filename %q, got %q", wantName, filename)
			}
		})
	}
}
