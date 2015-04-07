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

package stages

import (
	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/log"
	"github.com/coreos/ignition/registry"
)

type Stage interface {
	Run(config config.Config)
	Name() string
}

type StageCreator interface {
	Create(logger log.Logger, root string) Stage
	Name() string
}

var stages = registry.Create("stages")

func Register(stage StageCreator) {
	stages.Register(stage)
}

func Get(name string) StageCreator {
	s := stages.Get(name)
	if s == nil {
		return nil
	}
	return s.(StageCreator)
}

func Names() (names []string) {
	return stages.Names()
}
