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

package files

import (
	"fmt"

	"github.com/coreos/ignition/internal/config/types"
)

// createPasswd creates the users and groups as described in config.Passwd.
func (s stage) createPasswd(config types.Config) error {
	if err := s.createGroups(config); err != nil {
		return fmt.Errorf("failed to create groups: %v", err)
	}

	if err := s.createUsers(config); err != nil {
		return fmt.Errorf("failed to create users: %v", err)
	}

	return nil
}

// createUsers creates the users as described in config.Passwd.Users.
func (s stage) createUsers(config types.Config) error {
	if len(config.Passwd.Users) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createUsers")
	defer s.Logger.PopPrefix()

	for _, u := range config.Passwd.Users {
		if err := s.EnsureUser(u); err != nil {
			return fmt.Errorf("failed to create user %q: %v",
				u.Name, err)
		}

		if err := s.SetPasswordHash(u); err != nil {
			return fmt.Errorf("failed to set password for %q: %v",
				u.Name, err)
		}

		if err := s.AuthorizeSSHKeys(u); err != nil {
			return fmt.Errorf("failed to add keys to user %q: %v",
				u.Name, err)
		}
	}

	return nil
}

// createGroups creates the users as described in config.Passwd.Groups.
func (s stage) createGroups(config types.Config) error {
	if len(config.Passwd.Groups) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createGroups")
	defer s.Logger.PopPrefix()

	for _, g := range config.Passwd.Groups {
		if err := s.CreateGroup(g); err != nil {
			return fmt.Errorf("failed to create group %q: %v",
				g.Name, err)
		}
	}

	return nil
}
