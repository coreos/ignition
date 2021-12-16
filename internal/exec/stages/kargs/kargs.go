// Copyright 2021 Red Hat
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

package kargs

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/state"
)

const (
	name = "kargs"
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
			Fetcher: f,
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

func isNoOp(config types.Config) bool {
	return len(config.KernelArguments.ShouldExist) == 0 &&
		len(config.KernelArguments.ShouldNotExist) == 0
}

func (s stage) Apply(config types.Config, ignoreUnsupported bool) error {
	if isNoOp(config) || ignoreUnsupported {
		return nil
	}
	return errors.New("cannot apply kargs modifications live")
}

func (s stage) Run(config types.Config) error {
	if isNoOp(config) {
		return nil
	}

	if err := s.addKargs(config); err != nil {
		return fmt.Errorf("failed adding kernel arguments: %v", err)
	}

	return nil
}

func (s *stage) addKargs(config types.Config) error {
	var opts []string
	for _, arg := range config.KernelArguments.ShouldExist {
		opts = append(opts, "--should-exist", string(arg))
	}
	for _, arg := range config.KernelArguments.ShouldNotExist {
		opts = append(opts, "--should-not-exist", string(arg))
	}
	_, err := s.Logger.LogCmd(
		exec.Command(distro.KargsCmd(), opts...),
		"updating kernel arguments")
	return err
}
