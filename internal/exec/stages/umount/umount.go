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

package umount

import (
	"errors"
	"sort"

	cutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"

	"golang.org/x/sys/unix"
)

const (
	name = "umount"
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
	// n.b. sorted backwards
	sort.Slice(fss, func(i, j int) bool { return util.Depth(*fss[j].Path) < util.Depth(*fss[i].Path) })
	for _, fs := range fss {
		if err := s.umountFs(fs); err != nil {
			return err
		}
	}
	return nil
}

func (s stage) umountFs(fs types.Filesystem) error {
	if fs.Format == nil || *fs.Format == "swap" || *fs.Format == "" || *fs.Format == "none" {
		return nil
	}
	path, err := s.JoinPath(*fs.Path)
	if err != nil {
		return err
	}

	if err := s.Logger.LogOp(func() error { return unix.Unmount(path, 0) },
		"umounting %q", path,
	); err != nil {
		return err
	}
	return nil
}
