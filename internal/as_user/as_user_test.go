// Copyright 2026 Jason Colapietro
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

//go:build linux && cgo

package as_user_test

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"syscall"
	"testing"

	"github.com/coreos/ignition/v2/internal/as_user"
)

func TestOpenFileDropsSupplementaryGroups(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("test requires root")
	}

	const helperEnv = "IGNITION_TEST_SUPPLEMENTARY_GROUPS"
	if os.Getenv(helperEnv) != "1" {
		// Set the supplementary groups only in a subprocess so the test
		// runner's credentials cannot be changed by this test.
		cmd := exec.Command(os.Args[0], "-test.run=^TestOpenFileDropsSupplementaryGroups$", "-test.v")
		cmd.Env = append(os.Environ(), helperEnv+"=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: 0, Gid: 0, Groups: []uint32{4242}},
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("subprocess failed: %v\n%s", err, output)
		}
		return
	}

	// Unlike t.TempDir's parent directory, this directory can be traversed
	// by the unprivileged user after its permissions are changed.
	dir, err := os.MkdirTemp("", "ignition-as-user-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}

	u := &user.User{Uid: "65534", Gid: "65534"}
	for _, test := range []struct {
		name string
		uid  int
		gid  int
		mode os.FileMode
		want error
	}{
		{"supplementary group only", 0, 4242, 0070, syscall.EACCES},
		{"target user owned", 65534, 65534, 0700, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, test.name)
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chown(path, test.uid, test.gid); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, test.mode); err != nil {
				t.Fatal(err)
			}

			f, err := as_user.OpenFile(u, filepath.Join(path, "file"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if f != nil {
				if err := f.Close(); err != nil {
					t.Error(err)
				}
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("OpenFile returned %v, want %v", err, test.want)
			}
		})
	}

	groups, err := syscall.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(groups, []int{4242}) {
		t.Fatalf("OpenFile changed the caller's supplementary groups: %v", groups)
	}
}
