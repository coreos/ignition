// Copyright 2018 CoreOS, Inc.
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

package blackbox

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

var (
	testKillCounter   int
	testCancelled     bool
	testKilled        bool
	testCancelledLock sync.Mutex
	testChildSet      map[*exec.Cmd]struct{}
)

func init() {
	testChildSet = make(map[*exec.Cmd]struct{})
	signalChannel := make(chan os.Signal, 3)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			sig := <-signalChannel
			switch sig {
			case os.Interrupt, syscall.SIGTERM:
				if !testsAreCancelled() {
					fmt.Fprintf(os.Stderr, "SIGINT or SIGTERM received: cancelling remaining tests, press Ctrl+C twice more to immediately exit\n")
					cancelTheTests()
				} else if incrementTestKillCounter() {
					fmt.Fprintf(os.Stderr, "Exiting...\n")
					killTheTests()
				} else {
					fmt.Fprintf(os.Stderr, "press Ctrl+c once more to immediately exit\n")
				}
			}
		}
	}()
}

// testsAreCancelled returns if the tests have been cancelled.
// testsAreCancelled is thread safe.
func testsAreCancelled() bool {
	testCancelledLock.Lock()
	cancelled := testCancelled
	testCancelledLock.Unlock()
	return cancelled
}

// cancelTheTests marks the tests as cancelled, and future tests should be
// skipped. cancelTheTests is thread safe.
func cancelTheTests() {
	testCancelledLock.Lock()
	testCancelled = true
	testCancelledLock.Unlock()
	incrementTestKillCounter()
}

// incrementTestKillCounter increments a counter and returns true if the counter
// > 2. incrementTestKillCounter is thread safe.
func incrementTestKillCounter() bool {
	testCancelledLock.Lock()
	testKillCounter++
	ret := testKillCounter > 2
	testCancelledLock.Unlock()
	return ret
}

// killTheTests kills any currently running child processes via SIGKILL and
// calls os.Exit(1)
func killTheTests() {
	testCancelledLock.Lock()
	for cmd := range testChildSet {
		if cmd.Process != nil {
			// Processes started by the test framework are placed into a new
			// process group. Processes created by those processes should remain
			// in that new process group. The default process group id for a new
			// process is the same as its process id. Passing a negative number
			// to kill means the number represents a process group.
			//
			// Kill the inverse of the given process's pid to kill it and its
			// children.
			err := syscall.Kill(-1*cmd.Process.Pid, syscall.SIGKILL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "couldn't kill %q: %v\n", cmd.Path, err)
			}
		}
	}
	os.Exit(1)
}

// runCommandAndGetOutput will put the given process in the child process set,
// run the given command, remove it from the set, and then return its combined
// stdout/stderr and any errors. This should be used so that child processes can
// be killed if the tests are being uncleanly exited.
func runCommandAndGetOutput(cmd *exec.Cmd) ([]byte, error) {
	if cmd == nil {
		return nil, fmt.Errorf("cmd cannot be nil")
	}
	testCancelledLock.Lock()
	testChildSet[cmd] = struct{}{}
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	err := cmd.Start()
	if err != nil {
		testCancelledLock.Unlock()
		return nil, err
	}
	testCancelledLock.Unlock()

	err = cmd.Wait()

	testCancelledLock.Lock()
	delete(testChildSet, cmd)
	testCancelledLock.Unlock()

	return b.Bytes(), err
}
