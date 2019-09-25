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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_1_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
)

const (
	name = "files"

	// see https://github.com/systemd/systemd/commit/65e183d7899eb3725d3009196ac4decf1090b580
	relabelExtraDir = "/run/systemd/relabel-extra.d"
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
	if err := s.addRelabelUnit(); err != nil {
		return fmt.Errorf("failed to add relabel unit: %v", err)
	}

	// Add a file in /run/systemd/relabel-extra.d/ with paths that need to be relabeled
	// as early as possible (e.g. systemd units so systemd can read them while building its
	// graph). These are relabeled very early (right after policy load) so it cannot relabel
	// across mounts. Only relabel things in /etc here.
	if err := s.addRelabelExtraFile(); err != nil {
		return fmt.Errorf("failed to write systemd relabel file: %v", err)
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

	path, err := s.JoinPath(distro.RestoreconCmd())
	if err != nil {
		return fmt.Errorf("error resolving path for %s: %v", distro.RestoreconCmd(), err)
	}

	_, err = os.Lstat(path)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("targeting root without %s, cannot relabel", distro.RestoreconCmd())
	} else if err != nil {
		return fmt.Errorf("error checking for %s in root: %v", distro.RestoreconCmd(), err)
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
func (s *stage) addRelabelUnit() error {
	if len(s.toRelabel) == 0 {
		return nil
	}
	contents := `[Unit]
Description=Relabel files created by Ignition
DefaultDependencies=no
After=local-fs.target
Before=sysinit.target systemd-sysctl.service
ConditionSecurity=selinux
ConditionPathExists=/etc/selinux/ignition.relabel
OnFailure=emergency.target
OnFailureJobMode=replace-irreversibly

[Service]
Type=oneshot
ExecStart=` + distro.RestoreconCmd() + ` -0vRif /etc/selinux/ignition.relabel
ExecStart=/usr/bin/rm /etc/selinux/ignition.relabel
RemainAfterExit=yes`

	// create the unit file itself
	unit := types.Unit{
		Name:     "ignition-relabel.service",
		Contents: &contents,
	}

	if err := s.writeSystemdUnit(unit, true); err != nil {
		return err
	}

	if err := s.EnableRuntimeUnit(unit, "sysinit.target"); err != nil {
		return err
	}

	// and now create the list of files to relabel
	etcRelabelPath, err := s.JoinPath("etc/selinux/ignition.relabel")
	if err != nil {
		return err
	}
	f, err := os.Create(etcRelabelPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// yes, apparently the final \0 is needed
	_, err = f.WriteString(strings.Join(s.toRelabel, "\000") + "\000")
	return err
}

// addRelabelExtraFile writes a file to /run/systemd/relabel-extra.d/ with a list of files
// that should be relabeled immediately after policy load. In our case that's everything we
// wrote under /etc. This ensures systemd can access the files when building it's graph.
func (s stage) addRelabelExtraFile() error {
	relabelFilePath := filepath.Join(relabelExtraDir, "ignition.relabel")
	s.Logger.Info("adding relabel-extra.d/ file: %q", relabelFilePath)
	defer s.Logger.Info("finished adding relabel file")

	relabelFileContents := ""
	for _, file := range s.toRelabel {
		if strings.HasPrefix(file, "/etc") {
			relabelFileContents += file + "\n"
		}
	}
	if relabelFileContents == "" {
		return nil
	}
	if err := os.MkdirAll(relabelExtraDir, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(relabelFilePath, []byte(relabelFileContents), 0644)
}
