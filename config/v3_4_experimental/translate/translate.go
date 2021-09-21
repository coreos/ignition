// Copyright 2020 Red Hat, Inc.
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

package translate

import (
	"github.com/coreos/ignition/v2/config/translate"
	old_types "github.com/coreos/ignition/v2/config/v3_3/types"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
)

func translateIgnition(old old_types.Ignition) (ret types.Ignition) {
	// use a new translator so we don't recurse infinitely
	tr := translate.NewTranslator()
	tr.AddCustomTranslator(translateResource)
	tr.Translate(&old, &ret)
	ret.Version = types.MaxVersion.String()
	return
}

func translateResource(old old_types.Resource) (ret types.Resource) {
	tr := translate.NewTranslator()
	tr.Translate(&old.Compression, &ret.Compression)
	tr.Translate(&old.HTTPHeaders, &ret.HTTPHeaders)
	// Source: x is not equivalent to Sources: [x]
	tr.Translate(&old.Source, &ret.Source)
	tr.Translate(&old.Verification, &ret.Verification)
	return
}

func Translate(old old_types.Config) (ret types.Config) {
	tr := translate.NewTranslator()
	tr.AddCustomTranslator(translateResource)
	tr.AddCustomTranslator(translateIgnition)
	tr.Translate(&old, &ret)
	return
}
