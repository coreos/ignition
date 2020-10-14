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
	"path/filepath"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
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

	if err := s.createCrypttabEntries(config); err != nil {
		return fmt.Errorf("creating crypttab entries: %v", err)
	}

	if err := s.relabelFiles(); err != nil {
		return fmt.Errorf("failed to handle relabeling: %v", err)
	}

	return nil
}

// checkRelabeling determines whether relabeling is supported/requested so that
// we only collect filenames if we need to.
func (s *stage) checkRelabeling() error {
	if !distro.SelinuxRelabel() {
		s.Logger.Debug("compiled without relabeling support, skipping")
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
		for _, path := range paths {
			s.toRelabel = append(s.toRelabel, filepath.Join(s.DestDir, path))
		}
	}
}

// relabelFiles relabels all the files that were marked for relabeling using
// the libselinux APIs.
func (s *stage) relabelFiles() error {
	if s.toRelabel == nil || len(s.toRelabel) == 0 {
		return nil
	}

	// We could go further here and use the `setfscreatecon` API so that we
	// atomically create the files from the start with the right label, but (1)
	// atomicity isn't really necessary here since there is not even a policy
	// loaded and hence no MAC enforced, and (2) we'd still need after-the-fact
	// labeling for files created by processes we call out to, like `useradd`.

	return s.RelabelFiles(s.toRelabel)
}
