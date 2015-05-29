// Copyright 2015 CoreOS, Inc.
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
	"strconv"
	"strings"

	"github.com/coreos/ignition/config"
)

// CreateUser creates the user as described.
func (u Util) CreateUser(c config.User) error {
	if c.Create == nil {
		return nil
	}

	cu := c.Create
	args := []string{"--root", u.DestDir}

	if c.PasswordHash != "" {
		args = append(args, "--password", c.PasswordHash)
	} else {
		args = append(args, "--password", "*")
	}

	if cu.Uid != nil {
		args = append(args, "--uid",
			strconv.FormatUint(uint64(*cu.Uid), 10))
	}

	if cu.GECOS != "" {
		args = append(args, "--comment", fmt.Sprintf("%q", cu.GECOS))
	}

	if cu.Homedir != "" {
		args = append(args, "--home-dir", cu.Homedir)
	}

	if cu.NoCreateHome {
		args = append(args, "--no-create-home")
	} else {
		args = append(args, "--create-home")
	}

	if cu.PrimaryGroup != "" {
		args = append(args, "--gid", cu.PrimaryGroup)
	}

	if len(cu.Groups) > 0 {
		args = append(args, "--groups", strings.Join(cu.Groups, ","))
	}

	if cu.NoUserGroup {
		args = append(args, "--no-user-group")
	}

	if cu.System {
		args = append(args, "--system")
	}

	if cu.NoLogInit {
		args = append(args, "--no-log-init")
	}

	if cu.Shell != "" {
		args = append(args, "--shell", cu.Shell)
	}

	args = append(args, c.Name)

	return u.LogCmd(exec.Command("useradd", args...),
		"creating user %q", c.Name)
}

// Add the provided SSH public keys to the user's authorized keys.
func (u Util) AuthorizeSSHKeys(c config.User) error {
	if len(c.SSHAuthorizedKeys) == 0 {
		return nil
	}

	// TODO(vc): add the keys to the user
	return nil
}

// SetPasswordHash sets the password hash of the specified user.
func (u Util) SetPasswordHash(c config.User) error {
	if c.PasswordHash == "" {
		return nil
	}

	args := []string{
		"--root", u.DestDir,
		"--password", c.PasswordHash,
	}

	args = append(args, c.Name)

	return u.LogCmd(exec.Command("usermod", args...),
		"setting password for %q", c.Name)
}

// CreateGroup creates the group as described.
func (u Util) CreateGroup(g config.Group) error {
	args := []string{"--root", u.DestDir}

	if g.Gid != nil {
		args = append(args, "--gid",
			strconv.FormatUint(uint64(*g.Gid), 10))
	}

	if g.PasswordHash != "" {
		args = append(args, "--password", g.PasswordHash)
	} else {
		args = append(args, "--password", "*")
	}

	if g.System {
		args = append(args, "--system")
	}

	args = append(args, g.Name)

	return u.LogCmd(exec.Command("groupadd", args...),
		"adding group %q", g.Name)
}
