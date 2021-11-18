// Copyright 2021 Red Hat, Inc.
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

package util

import (
	"fmt"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

// runGetExit runs the command and returns the exit status. It only returns an error when execing
// the command encounters an error. exec'd programs that exit with non-zero status will not return
// errors.
func runGetExit(cmd string, args ...string) (int, string, error) {
	tmp, err := exec.Command(cmd, args...).CombinedOutput()
	logs := string(tmp)
	if err == nil {
		return 0, logs, nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return -1, logs, err
	}
	status := exitErr.ExitCode()
	return status, logs, nil
}

func UmountPath(path string) error {
	// Retry a few times on unmount failure, checking each time whether
	// the umount actually succeeded but claimed otherwise.  See
	// https://github.com/coreos/bootengine/commit/8bf46fe78ec5 for more
	// context.
	var unmountErr error
	for i := 0; i < 3; i++ {
		if unmountErr = unix.Unmount(path, 0); unmountErr == nil {
			return nil
		}

		// wait a sec to see if things clear up
		time.Sleep(time.Second)

		if unmounted, _, err := runGetExit("mountpoint", "-q", path); err != nil {
			return fmt.Errorf("exec'ing `mountpoint -q %s` failed: %v", path, err)
		} else if unmounted == 1 {
			return nil
		}
	}
	return fmt.Errorf("umount failed after 3 tries for %s: %w", path, unmountErr)
}
