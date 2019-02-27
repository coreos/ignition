// Copyright 2018 CoreOS, Inc.
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

package config

import (
	"github.com/coreos/ignition/config/translate"
	from "github.com/coreos/ignition/config/v3_0_experimental/types"
	"github.com/coreos/ignition/internal/config/types"
)

var (
	tr translate.Translator
)

func translateIgnition(old from.Ignition) types.Ignition {
	ret := types.Ignition{}
	tr.Translate(&old.Config, &ret.Config)
	tr.Translate(&old.Timeouts, &ret.Timeouts)
	tr.Translate(&old.Security, &ret.Security)
	ret.Version = types.MaxVersion.String()

	return ret
}

func Translate(old from.Config) types.Config {
	ret := types.Config{}
	tr = translate.NewTranslator()
	tr.AddCustomTranslator(translateIgnition)
	tr.Translate(&old, &ret)
	return ret
}
