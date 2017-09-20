// Copyright 2017 CoreOS, Inc.
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

package noop

import (
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/exec/stages"
	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/resource"
)

const (
	name = "noop"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(_ *log.Logger, _ string, _ resource.Fetcher) stages.Stage {
	return &stage{}
}

func (creator) Name() string {
	return name
}

type stage struct{}

func (stage) Name() string {
	return name
}

func (s stage) Run(_ types.Config) bool {
	return true
}
