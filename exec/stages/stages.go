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
	"fmt"

	"github.com/coreos/ignition/config"
	"github.com/coreos/ignition/log"
)

type Stage interface {
	Run(config config.Config)
	Name() string
}

type StageCreator interface {
	Create(logger log.Logger, root string) Stage
	Name() string
}

var stages map[string]StageCreator

func Register(stage StageCreator) {
	if stages == nil {
		stages = map[string]StageCreator{}
	}
	if _, ok := stages[stage.Name()]; ok {
		panic(fmt.Sprintf("stage %q already registered", stage.Name()))
	}
	stages[stage.Name()] = stage
}

func Get(name string) StageCreator {
	if stage, ok := stages[name]; ok {
		return stage
	}

	return nil
}

func Names() (names []string) {
	for name := range stages {
		names = append(names, name)
	}
	return
}
