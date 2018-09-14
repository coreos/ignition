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

package files

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/ignition/internal/config/types"
	"github.com/coreos/ignition/internal/distro"
	"github.com/coreos/ignition/internal/exec/stages"
	"github.com/coreos/ignition/internal/exec/util"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/resource"
)

const (
	name = "files"
)

var (
	ErrFilesystemUndefined = errors.New("the referenced filesystem was not defined")
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, root string, f resource.Fetcher) stages.Stage {
	return &stage{
		Util: util.Util{
			DestDir: root,
			Logger:  logger,
			Fetcher: f,
		},
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	util.Util
	toRelabel []string
}

func (stage) Name() string {
	return name
}

func (s stage) Run(config types.Config) error {
	if err := s.checkRelabeling(); err != nil {
		return fmt.Errorf("failed to check if SELinux labeling required: %v", err)
	}

	if err := s.createPasswd(config); err != nil {
		return fmt.Errorf("failed to create users/groups: %v", err)
	}

	if err := s.createFilesystemsEntries(config); err != nil {
		return fmt.Errorf("failed to create files: %v", err)
	}

	if err := s.createUnits(config); err != nil {
		return fmt.Errorf("failed to create units: %v", err)
	}

	// add systemd unit to relabel files
	if err := s.addRelabelUnit(config); err != nil {
		return fmt.Errorf("failed to add relabel unit: %v", err)
	}

	return nil
}

// checkRelabeling determines whether relabeling is supported/requested so that
// we only collect filenames if we need to.
func (s *stage) checkRelabeling() error {
	if !distro.SelinuxRelabel() || distro.RestoreconCmd() == "" {
		s.Logger.Debug("compiled without relabeling support, skipping")
		return nil
	}

	exists, err := s.PathExists(distro.RestoreconCmd())
	if err != nil {
		return err
	} else if !exists {
		s.Logger.Debug("targeting root without %s, skipping relabel", distro.RestoreconCmd())
		return nil
	}

	// initialize to non-nil (whereas a nil slice means not to append, even
	// though they're functionally equivalent)
	s.toRelabel = []string{}
	return nil
}

// relabeling returns true if we are relabeling, false otherwise.
func (s *stage) relabeling() bool {
	return s.toRelabel != nil
}

// relabel adds one or more paths to the list of paths that need relabeling.
func (s *stage) relabel(paths ...string) {
	if s.toRelabel != nil {
		s.toRelabel = append(s.toRelabel, paths...)
	}
}

// addRelabelUnit creates and enables a runtime systemd unit to run restorecon
// if there are files that need to be relabeled.
func (s *stage) addRelabelUnit(config types.Config) error {
	if s.toRelabel == nil || len(s.toRelabel) == 0 {
		return nil
	}

	// create the unit file itself
	unit := types.Unit{
		Name: "ignition-relabel.service",
		Contents: `[Unit]
Description=Relabel files created by Ignition
DefaultDependencies=no
After=local-fs.target
Before=sysinit.target
ConditionSecurity=selinux
ConditionPathExists=/etc/selinux/ignition.relabel
OnFailure=emergency.target
OnFailureJobMode=replace-irreversibly

[Service]
Type=oneshot
ExecStart=` + distro.RestoreconCmd() + ` -0vRif /etc/selinux/ignition.relabel
ExecStart=/usr/bin/rm /etc/selinux/ignition.relabel
RemainAfterExit=yes`,
	}

	if err := s.writeSystemdUnit(unit, true); err != nil {
		return err
	}

	if err := s.EnableRuntimeUnit(unit, "sysinit.target"); err != nil {
		return err
	}

	// and now create the list of files to relabel
	f, err := os.Create(s.JoinPath("etc/selinux/ignition.relabel"))
	if err != nil {
		return err
	}
	defer f.Close()

	// yes, apparently the final \0 is needed
	_, err = f.WriteString(strings.Join(s.toRelabel, "\000") + "\000")
	return err
}
