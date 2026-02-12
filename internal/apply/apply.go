// Copyright 2021 Red Hat, Inc.
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

package apply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos/ignition/v2/internal/exec"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/disks"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/fetch"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/fetch_offline"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/files"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/kargs"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/mount"
	_ "github.com/coreos/ignition/v2/internal/exec/stages/umount"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"
	"github.com/coreos/ignition/v2/internal/util"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
)

type Flags struct {
	Root              string
	IgnoreUnsupported bool
	Offline           bool
}

func inContainer() bool {
	// this is set by various container tools like systemd-nspawn, podman,
	// docker, etc...
	if val, _ := os.LookupEnv("container"); val != "" {
		return true
	}

	// check for the podman or docker container files
	paths := []string{"/run/.containerenv", "/.dockerenv"}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

func Run(cfg types.Config, flags Flags, logger *log.Logger) error {
	if !inContainer() {
		return errors.New("this tool is not designed to run on a host system; reprovision the machine instead")
	}

	// make absolute because our code assumes that
	var err error
	if flags.Root, err = filepath.Abs(flags.Root); err != nil {
		return err
	}

	fetcher := resource.Fetcher{
		Logger:  logger,
		Offline: flags.Offline,
	}

	state := state.State{}
	cfgFetcher := exec.ConfigFetcher{
		Logger:  logger,
		Fetcher: &fetcher,
		State:   &state,
	}

	finalCfg, err := cfgFetcher.RenderConfig(cfg)
	if err != nil {
		return err
	}

	// verify upfront if we'll need networking but we're not allowed
	if flags.Offline {
		stage := stages.Get("fetch-offline").Create(logger, flags.Root, fetcher, &state)
		if err := stage.Run(finalCfg); err != nil {
			return err
		}
	}

	// Order in which to apply live. This is overkill since effectively only
	// `files` supports it right now, but let's be extensible. Also ensures that
	// all stages are accounted for.
	stagesOrder := []string{"fetch-offline", "fetch", "kargs", "disks", "mount", "files", "umount"}
	allStages := stages.Names()
	if len(stagesOrder) != len(allStages) {
		panic(fmt.Sprintf("%v != %v", stagesOrder, allStages))
	}

	for _, stageName := range stagesOrder {
		if !util.StrSliceContains(allStages, stageName) {
			panic(fmt.Sprintf("stage '%s' invalid", stageName))
		}
		stage := stages.Get(stageName).Create(logger, flags.Root, fetcher, &state)
		if err := stage.Apply(finalCfg, flags.IgnoreUnsupported); err != nil {
			return fmt.Errorf("running stage '%s': %w", stageName, err)
		}
	}

	return nil
}
