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

package distro

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPartitionerFromCmdline(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		cmdline     string
		writeFile   bool
		wantBackend string
		wantWarning bool
	}{
		{
			name:        "env sfdisk",
			env:         "sfdisk",
			wantBackend: "sfdisk",
		},
		{
			name:        "env sgdisk",
			env:         "sgdisk",
			wantBackend: "sgdisk",
		},
		{
			name:        "invalid env falls through to cmdline",
			env:         "bogus",
			cmdline:     "root=/dev/sda1 ignition.partitioner=sfdisk",
			writeFile:   true,
			wantBackend: "sfdisk",
		},
		{
			name:        "cmdline sfdisk",
			cmdline:     "ignition.partitioner=sfdisk",
			writeFile:   true,
			wantBackend: "sfdisk",
		},
		{
			name:        "cmdline sgdisk",
			cmdline:     "ignition.partitioner=sgdisk",
			writeFile:   true,
			wantBackend: "sgdisk",
		},
		{
			name:        "cmdline unrecognized value warns and defaults",
			cmdline:     "ignition.partitioner=bogus",
			writeFile:   true,
			wantBackend: "sgdisk",
			wantWarning: true,
		},
		{
			name:        "later valid value wins over earlier invalid one",
			cmdline:     "ignition.partitioner=bogus ignition.partitioner=sfdisk",
			writeFile:   true,
			wantBackend: "sfdisk",
		},
		{
			name:        "no partitioner argument",
			cmdline:     "root=/dev/sda1 console=ttyS0",
			writeFile:   true,
			wantBackend: "sgdisk",
		},
		{
			name:        "argument without value",
			cmdline:     "ignition.partitioner",
			writeFile:   true,
			wantBackend: "sgdisk",
		},
		{
			name:        "unreadable cmdline warns and defaults",
			wantBackend: "sgdisk",
			wantWarning: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IGNITION_PARTITIONER", tt.env)
			path := filepath.Join(t.TempDir(), "cmdline")
			if tt.writeFile {
				if err := os.WriteFile(path, []byte(tt.cmdline), 0644); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("IGNITION_KERNEL_CMDLINE_PATH", path)

			backend, err := readPartitionerFromCmdline()
			if backend != tt.wantBackend {
				t.Errorf("backend: want %q, got %q", tt.wantBackend, backend)
			}
			if gotWarning := err != nil; gotWarning != tt.wantWarning {
				t.Errorf("warning: want %v, got %v (%v)", tt.wantWarning, gotWarning, err)
			}
		})
	}
}
