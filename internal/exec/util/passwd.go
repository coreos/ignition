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
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/coreos/ignition/config/v3_0_experimental/types"
	"github.com/coreos/ignition/internal/as_user"
	"github.com/coreos/ignition/internal/distro"
	"github.com/coreos/ignition/internal/log"
)

// EnsureUser ensures that the user exists as described. If the user does not
// yet exist, they will be created, otherwise the existing user will be
// modified.
func (u Util) EnsureUser(c types.PasswdUser) error {
	exists, err := u.CheckIfUserExists(c)
	if err != nil {
		return err
	}
	args := []string{"--root", u.DestDir}

	var cmd string
	if exists {
		cmd = distro.UsermodCmd()

		if c.HomeDir != "" {
			args = append(args, "--home", c.HomeDir, "--move-home")
		}
	} else {
		cmd = distro.UseraddCmd()

		if c.HomeDir != "" {
			args = append(args, "--home-dir", c.HomeDir)
		}

		if c.NoCreateHome {
			args = append(args, "--no-create-home")
		} else {
			args = append(args, "--create-home")
		}

		if c.NoUserGroup {
			args = append(args, "--no-user-group")
		}

		if c.System {
			args = append(args, "--system")
		}

		if c.NoLogInit {
			args = append(args, "--no-log-init")
		}
	}

	if c.PasswordHash != nil {
		if *c.PasswordHash != "" {
			args = append(args, "--password", *c.PasswordHash)
		} else {
			args = append(args, "--password", "*")
		}
	} else if !exists {
		// Set the user's password to "*" if they don't exist yet and one wasn't
		// set to disable password logins
		args = append(args, "--password", "*")
	}

	if c.UID != nil {
		args = append(args, "--uid",
			strconv.FormatUint(uint64(*c.UID), 10))
	}

	if c.Gecos != "" {
		args = append(args, "--comment", c.Gecos)
	}

	if c.PrimaryGroup != "" {
		args = append(args, "--gid", c.PrimaryGroup)
	}

	if len(c.Groups) > 0 {
		args = append(args, "--groups", strings.Join(translateV2_1PasswdUserGroupSliceToStringSlice(c.Groups), ","))
	}

	if c.Shell != "" {
		args = append(args, "--shell", c.Shell)
	}

	args = append(args, c.Name)

	_, err = u.LogCmd(exec.Command(cmd, args...),
		"creating or modifying user %q", c.Name)
	return err
}

// CheckIfUserExists will return Info log when user is empty
func (u Util) CheckIfUserExists(c types.PasswdUser) (bool, error) {
	code := -1
	cmd := exec.Command(distro.ChrootCmd(), u.DestDir, distro.IdCmd(), c.Name)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.Sys().(syscall.WaitStatus).ExitStatus()
		}
		if code == 1 {
			u.Info("checking if user \"%s\" exists: %s", c.Name, fmt.Errorf("[Attention] %v: Cmd: %s Stdout: %s", err, log.QuotedCmd(cmd), stdout))
			return false, nil
		}
		u.Logger.Info("error encountered (%T): %v", err, err)
		return false, err
	}
	return true, nil
}

// golang--
func translateV2_1PasswdUserGroupSliceToStringSlice(groups []types.Group) []string {
	newGroups := make([]string, len(groups))
	for i, g := range groups {
		newGroups[i] = string(g)
	}
	return newGroups
}

// writeAuthKeysFile writes the content in keys to the path fp for the user,
// creating any directories in fp as needed.
func writeAuthKeysFile(u *user.User, fp string, keys []byte) error {
	if err := as_user.MkdirAll(u, filepath.Dir(fp), 0700); err != nil {
		return err
	}

	f, err := as_user.OpenFile(u, fp, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(keys); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

// AuthorizeSSHKeys adds the provided SSH public keys to the user's authorized keys.
func (u Util) AuthorizeSSHKeys(c types.PasswdUser) error {
	if len(c.SSHAuthorizedKeys) == 0 {
		return nil
	}

	return u.LogOp(func() error {
		usr, err := u.userLookup(c.Name)
		if err != nil {
			return fmt.Errorf("unable to lookup user %q", c.Name)
		}

		// TODO(vc): introduce key names to config?
		// TODO(vc): validate c.SSHAuthorizedKeys well-formedness.
		ks := strings.Join(translateV2_1SSHAuthorizedKeySliceToStringSlice(c.SSHAuthorizedKeys), "\n")
		// XXX(vc): for now ensure the addition is always
		// newline-terminated.  A future version of akd will handle this
		// for us in addition to validating the ssh keys for
		// well-formedness.
		if !strings.HasSuffix(ks, "\n") {
			ks = ks + "\n"
		}

		if distro.WriteAuthorizedKeysFragment() {
			writeAuthKeysFile(usr, filepath.Join(usr.HomeDir, ".ssh", "authorized_keys.d", "ignition"), []byte(ks))
		} else {
			writeAuthKeysFile(usr, filepath.Join(usr.HomeDir, ".ssh", "authorized_keys"), []byte(ks))
		}

		return nil
	}, "adding ssh keys to user %q", c.Name)
}

// golang--
func translateV2_1SSHAuthorizedKeySliceToStringSlice(keys []types.SSHAuthorizedKey) []string {
	newKeys := make([]string, len(keys))
	for i, k := range keys {
		newKeys[i] = string(k)
	}
	return newKeys
}

// SetPasswordHash sets the password hash of the specified user.
func (u Util) SetPasswordHash(c types.PasswdUser) error {
	if c.PasswordHash == nil {
		return nil
	}

	pwhash := *c.PasswordHash
	if *c.PasswordHash == "" {
		pwhash = "*"
	}

	args := []string{
		"--root", u.DestDir,
		"--password", pwhash,
	}

	args = append(args, c.Name)

	_, err := u.LogCmd(exec.Command(distro.UsermodCmd(), args...),
		"setting password for %q", c.Name)
	return err
}

// CreateGroup creates the group as described.
func (u Util) CreateGroup(g types.PasswdGroup) error {
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

	_, err := u.LogCmd(exec.Command(distro.GroupaddCmd(), args...),
		"adding group %q", g.Name)
	return err
}
