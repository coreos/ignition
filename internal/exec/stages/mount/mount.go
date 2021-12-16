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

// The storage stage is responsible for partitioning disks, creating RAID
// arrays, formatting partitions, writing files, writing systemd units, and
// writing network units.

package mount

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"
)

const (
	name = "mount"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, root string, f resource.Fetcher, state *state.State) stages.Stage {
	return &stage{
		Util: util.Util{
			DestDir: root,
			Logger:  logger,
			State:   state,
		},
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	util.Util
}

func (stage) Name() string {
	return name
}

func (s stage) Apply(config types.Config, ignoreUnsupported bool) error {
	if len(config.Storage.Filesystems) == 0 || ignoreUnsupported {
		return nil
	}
	return errors.New("cannot apply filesystems modifications live")
}

func (s stage) Run(config types.Config) error {
	fss := []types.Filesystem{}
	for _, fs := range config.Storage.Filesystems {
		if cutil.NotEmpty(fs.Path) {
			fss = append(fss, fs)
		}
	}
	sort.Slice(fss, func(i, j int) bool { return util.Depth(*fss[i].Path) < util.Depth(*fss[j].Path) })
	for _, fs := range fss {
		if err := s.mountFs(fs); err != nil {
			return err
		}
	}
	return nil
}

// checkForNonDirectories returns an error if any element of path is not a directory
func checkForNonDirectories(path string) error {
	p := "/"
	for _, component := range util.SplitPath(path) {
		p = filepath.Join(p, component)
		st, err := os.Lstat(p)
		if err != nil && os.IsNotExist(err) {
			return nil // nonexistent is ok
		} else if err != nil {
			return err
		}
		if !st.Mode().IsDir() {
			return fmt.Errorf("mount path %q contains non-directory component %q", path, p)
		}
	}
	return nil
}

func (s stage) mountFs(fs types.Filesystem) error {
	if fs.Format == nil || *fs.Format == "swap" || *fs.Format == "" || *fs.Format == "none" {
		return nil
	}

	// mount paths shouldn't include symlinks or other non-directories so we can use filepath.Join()
	// instead of s.JoinPath(). Check that the resulting path is composed of only directories.
	relpath := *fs.Path
	path := filepath.Join(s.DestDir, relpath)
	if err := checkForNonDirectories(path); err != nil {
		return err
	}

	var firstMissing string
	if distro.SelinuxRelabel() {
		var err error
		firstMissing, err = util.FindFirstMissingPathComponent(path)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		// Record created directories for use by the files stage.
		// NotateMkdirAll() is relative to the DestDir.
		if err := s.NotateMkdirAll(relpath, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if distro.SelinuxRelabel() {
		if err := s.RelabelFiles([]string{firstMissing}); err != nil {
			return err
		}
	}

	args := translateOptionSliceToString(fs.MountOptions, ",")
	cmd := exec.Command(distro.MountCmd(), "-o", args, "-t", *fs.Format, fs.Device, path)
	if _, err := s.Logger.LogCmd(cmd,
		"mounting %q at %q with type %q and options %q", fs.Device, path, *fs.Format, args,
	); err != nil {
		return err
	}

	if distro.SelinuxRelabel() {
		// relabel the root of the disk if it's fresh
		if isEmpty, err := util.FilesystemIsEmpty(path); err != nil {
			return fmt.Errorf("Checking if mountpoint %s is empty: %v", path, err)
		} else if isEmpty {
			if err := s.RelabelFiles([]string{path}); err != nil {
				return err
			}
		}
	}

	return nil
}

func translateOptionSliceToString(opts []types.MountOption, separator string) string {
	mountOpts := make([]string, len(opts))
	for i, o := range opts {
		mountOpts[i] = string(o)
	}
	return strings.Join(mountOpts, separator)
}
