// Copyright 2019 Red Hat, Inc.
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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/coreos/ignition/v2/internal/distro"
)

const (
	selinuxConfig       = "/etc/selinux/config"
	selinuxFileContexts = "contexts/files/file_contexts"
)

var selinuxPolicy = ""

func (ut Util) getSelinuxPolicy() (string, error) {
	if selinuxPolicy == "" {
		configPath, err := ut.JoinPath(selinuxConfig)
		if err != nil {
			return "", err
		}

		file, err := os.Open(configPath)
		if err != nil {
			return "", fmt.Errorf("failed to open %v: %v", selinuxConfig, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "SELINUXTYPE=") {
				policy := line[len("SELINUXTYPE="):]
				if len(policy) == 0 {
					return "", fmt.Errorf("invalid SELINUXTYPE value in %v", selinuxConfig)
				}
				selinuxPolicy = policy
				break
			}
		}

		if selinuxPolicy == "" {
			return "", fmt.Errorf("didn't find SELINUXTYPE in %v", selinuxConfig)
		}
	}

	return selinuxPolicy, nil
}

// RelabelFiles relabels all the files matching the globby patterns given.
func (ut Util) RelabelFiles(patterns []string) error {
	policy, err := ut.getSelinuxPolicy()
	if err != nil {
		return err
	}

	file_contexts, err := ut.JoinPath("/etc/selinux", policy, selinuxFileContexts)
	if err != nil {
		return err
	}

	cmd := exec.Command(distro.SetfilesCmd(), "-vF0", "-r", ut.DestDir, file_contexts, "-f", "-")
	cmd.Stdin = strings.NewReader(strings.Join(patterns, "\000") + "\000")
	if _, err := ut.Logger.LogCmd(cmd, "relabeling %d patterns", len(patterns)); err != nil {
		return err
	}
	return nil
}
