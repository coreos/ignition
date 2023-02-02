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
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/journal"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/as_user"
	"github.com/coreos/ignition/v2/internal/distro"
	"golang.org/x/sys/unix"
)

const (
	// ignitionSSHAuthorizedkeysMessageID keeps track of the journald
	// log related to ssh_authorized_keys information.
	ignitionSSHAuthorizedkeysMessageID = "225067b87bbd4a0cb6ab151f82fa364b"
)

func appendIfTrue(args []string, test *bool, newargs string) []string {
	if util.IsTrue(test) {
		return append(args, newargs)
	}
	return args
}

func appendIfStringSet(args []string, arg string, str *string) []string {
	if util.NotEmpty(str) {
		return append(args, arg, *str)
	}
	return args
}

// EnsureUser ensures that the user exists as described. If the user does not
// yet exist, they will be created, otherwise the existing user will be modified.
// If the `shouldExist` field is set to false and the user already exists, then
// they will be deleted.
func (u Util) EnsureUser(c types.PasswdUser) error {
	shouldExist := !util.IsFalse(c.ShouldExist)
	exists, err := u.CheckIfUserExists(c)
	if err != nil {
		return err
	}
	if !shouldExist {
		if exists {
			args := []string{"--remove", "--root", u.DestDir, c.Name}
			_, err := u.LogCmd(exec.Command(distro.UserdelCmd(), args...),
				"deleting user %q", c.Name)
			if err != nil {
				return fmt.Errorf("failed to delete user %q: %v",
					c.Name, err)
			}
		}
		return nil
	}

	args := []string{"--root", u.DestDir}

	var cmd string
	if exists {
		cmd = distro.UsermodCmd()

		if util.NotEmpty(c.HomeDir) {
			args = append(args, "--home", *c.HomeDir, "--move-home")
		}
	} else {
		cmd = distro.UseraddCmd()

		args = appendIfStringSet(args, "--home-dir", c.HomeDir)

		if util.IsTrue(c.NoCreateHome) {
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

// GetUserHomeDir returns the user home directory. Note that DestDir is not
// prefixed.
func (u Util) GetUserHomeDir(c types.PasswdUser) (string, error) {
	homedir, err := u.GetUserHomeDirByName(c.Name)
	if err != nil {
		return "", err
	}
	return homedir, nil
}

// GetUserHomeDirByName returns the user home directory. Note that DestDir is not
// prefixed.
func (u Util) GetUserHomeDirByName(name string) (string, error) {
	usr, err := u.userLookup(name)
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
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
		return fmt.Errorf("creating parent dirs for %q: %w", fp, err)
	}

	f, err := as_user.OpenFile(u, fp, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening file %q as user %s and group %s: %w", fp, u.Uid, u.Gid, err)
	}
	if _, err = f.Write(keys); err != nil {
		f.Close() // ignore errors
		return fmt.Errorf("writing file %q: %w", fp, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing file %q: %w", fp, err)
	}
	return nil
}

// ModifyHomeDirPermissions changes the permissions for the user's home
// directory if it was created during the Ignition mount stage as a
// result of a mountpoint.
func (u Util) ModifyHomeDirPermissions(c types.PasswdUser) error {
	if util.IsTrue(c.NoCreateHome) {
		return nil
	}

	homeDir, err := u.GetUserHomeDir(c)
	if err != nil {
		return err
	}

	usr, err := u.userLookup(c.Name)
	if err != nil {
		return fmt.Errorf("unable to look up user %q: %w", c.Name, err)
	}
	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return fmt.Errorf("converting uid to int: %w", err)
	}
	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return fmt.Errorf("converting gid to int: %w", err)
	}

	for _, dir := range u.State.NotatedDirectories {
		if dir == homeDir || strings.HasPrefix(dir, homeDir+"/") {
			dir, err = u.JoinPath(dir)
			if err != nil {
				return fmt.Errorf("calculating path of %q: %w", dir, err)
			}
			if err := os.Chown(dir, uid, gid); err != nil {
				return fmt.Errorf("changing ownership of %q: %w", dir, err)
			}
			// more restrictive than necessary for some subdirs,
			// but default to secure
			if err := os.Chmod(dir, 0700); err != nil {
				return fmt.Errorf("changing file mode of %q: %w", dir, err)
			}
		}
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
		var path string
		if distro.WriteAuthorizedKeysFragment() {
			path, err = u.JoinPath(usr.HomeDir, ".ssh", "authorized_keys.d", "ignition")
			if err == nil {
				err = writeAuthKeysFile(usr, path, []byte(ks))
			}
		} else {
			path, err = u.JoinPath(usr.HomeDir, ".ssh", "authorized_keys")
			if err == nil {
				err = writeAuthKeysFile(usr, path, []byte(ks))
			}
		}
		if err != nil {
			return fmt.Errorf("failed to set SSH key: %v", err)
		}
		_ = journal.Send(fmt.Sprintf("wrote ssh authorized keys file for user: %s", c.Name), journal.PriInfo, map[string]string{
			"IGNITION_USER_NAME": c.Name,
			"IGNITION_PATH":      path,
			"MESSAGE_ID":         ignitionSSHAuthorizedkeysMessageID,
		})
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

// EnsureGroup ensures that the group exists as described. If the
// `shouldExist` field is set to false and the group already exists,
// then it will be deleted.
func (u Util) EnsureGroup(g types.PasswdGroup) error {
	shouldExist := !util.IsFalse(g.ShouldExist)
	exists, err := u.CheckIfGroupExists(g)
	if err != nil {
		return err
	}
	if !shouldExist {
		if exists {
			args := []string{"--root", u.DestDir, g.Name}
			_, err := u.LogCmd(exec.Command(distro.GroupdelCmd(), args...),
				"deleting group %q", g.Name)
			if err != nil {
				return fmt.Errorf("failed to delete group %q: %v",
					g.Name, err)
			}
		}
		return nil
	}

	args := []string{"--root", u.DestDir}

	if g.Gid != nil {
		args = append(args, "--gid",
			strconv.FormatUint(uint64(*g.Gid), 10))
	}

	if util.NotEmpty(g.PasswordHash) {
		args = append(args, "--password", *g.PasswordHash)
	} else {
		args = append(args, "--password", "*")
	}

	args = appendIfTrue(args, g.System, "--system")

	args = append(args, g.Name)

	_, err = u.LogCmd(exec.Command(distro.GroupaddCmd(), args...),
		"adding group %q", g.Name)
	return err
}

// CheckIfGroupExists will return Info log when group is empty
func (u Util) CheckIfGroupExists(g types.PasswdGroup) (bool, error) {
	_, err := u.groupLookup(g.Name)
	if _, ok := err.(user.UnknownGroupError); ok {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
