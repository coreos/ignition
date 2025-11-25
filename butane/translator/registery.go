// Copyright 2022 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translator

import (
	"fmt"

	"github.com/coreos/butane/config/common"
	"github.com/coreos/vcontext/report"
	"gopkg.in/yaml.v3"
)

var TranslatorRegistry = &Registry{
	translators: map[string]Translator{},
}

type Registry struct {
	translators map[string]Translator
}

func (r *Registry) RegisterTranslator(trans Translator) {
	cf := trans.Metadata().commonFields
	if _, ok := r.translators[cf.asKey()]; ok {
		panic(fmt.Sprintf("tried to reregister existing translator (%+v)", trans.Metadata()))
	}
	r.translators[cf.asKey()] = trans
}

func (r *Registry) TranslateBytes(input []byte, options common.TranslateBytesOptions) ([]byte, report.Report, error) {
	// first determine version; this will ignore most fields
	cf := commonFields{}
	if err := yaml.Unmarshal(input, &cf); err != nil {
		return nil, report.Report{}, common.ErrUnmarshal{
			Detail: err.Error(),
		}
	}

	translator := r.translators[cf.asKey()]
	return translator.TranslateBytes(input, options)
}
