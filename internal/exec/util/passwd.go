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

	"github.com/coreos/ignition/v2/config/v3_1_experimental/types"
	"github.com/coreos/ignition/v2/internal/as_user"
	"github.com/coreos/ignition/v2/internal/distro"
	"golang.org/x/sys/unix"
)

func appendIfTrue(args []string, test *bool, newargs string) []string {
	if test != nil && *test {
		return append(args, newargs)
	}
	return args
}

func appendIfStringSet(args []string, arg string, str *string) []string {
	if str != nil && *str != "" {
		return append(args, arg, *str)
	}
	return args
}

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

		if c.HomeDir != nil && *c.HomeDir != "" {
			args = append(args, "--home", *c.HomeDir, "--move-home")
		}
	} else {
		cmd = distro.UseraddCmd()

		args = appendIfStringSet(args, "--home-dir", c.HomeDir)

		if c.NoCreateHome != nil && *c.NoCreateHome {
			args = append(args, "--no-create-home")
		} else {
			args = append(args, "--create-home")
		}

		args = appendIfTrue(args, c.NoUserGroup, "--no-user-group")
		args = appendIfTrue(args, c.System, "--system")
		args = appendIfTrue(args, c.NoLogInit, "--no-log-init")
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

	args = appendIfStringSet(args, "--comment", c.Gecos)
	args = appendIfStringSet(args, "--gid", c.PrimaryGroup)

	if len(c.Groups) > 0 {
		args = append(args, "--groups", strings.Join(translateV2_1PasswdUserGroupSliceToStringSlice(c.Groups), ","))
	}

	args = appendIfStringSet(args, "--shell", c.Shell)

	args = append(args, c.Name)

	_, err = u.LogCmd(exec.Command(cmd, args...),
		"creating or modifying user %q", c.Name)
	return err
}

// CheckIfUserExists will return Info log when user is empty
func (u Util) CheckIfUserExists(c types.PasswdUser) (bool, error) {
	_, err := u.userLookup(c.Name)
	if _, ok := err.(user.UnknownUserError); ok {
		return false, nil
	}
	if err != nil {
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

	f, err := as_user.OpenFile(u, fp, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC, 0600)
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
			err = writeAuthKeysFile(usr, filepath.Join(usr.HomeDir, ".ssh", "authorized_keys.d", "ignition"), []byte(ks))
		} else {
			err = writeAuthKeysFile(usr, filepath.Join(usr.HomeDir, ".ssh", "authorized_keys"), []byte(ks))
		}

		if err != nil {
			return fmt.Errorf("failed to set SSH key: %v", err)
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

	if g.PasswordHash != nil && *g.PasswordHash != "" {
		args = append(args, "--password", *g.PasswordHash)
	} else {
		args = append(args, "--password", "*")
	}

	args = appendIfTrue(args, g.System, "--system")

	args = append(args, g.Name)

	_, err := u.LogCmd(exec.Command(distro.GroupaddCmd(), args...),
		"adding group %q", g.Name)
	return err
}
