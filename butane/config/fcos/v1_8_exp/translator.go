// Copyright 2026 Red Hat, Inc
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
// limitations under the License.)

package v1_8_exp

import (
	"fmt"

	"github.com/coreos/ignition/v2/butane/config/common"
	cutil "github.com/coreos/ignition/v2/butane/config/util"
	"github.com/coreos/ignition/v2/butane/translator"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/vcontext/report"
	"github.com/coreos/vcontext/tree"
)

type specTranslator struct{}

type parsedConfig struct {
	config      Config
	contextTree tree.Node
}

var _ translator.Translator = specTranslator{}

func init() {
	translator.Global.Register(specTranslator{})
}

func (specTranslator) Metadata() translator.Metadata {
	return translator.Metadata{
		Variant: "fcos",
		Version: semver.Version{
			Major:      1,
			Minor:      8,
			PreRelease: "experimental",
		},
		Description:     "Fedora CoreOS",
		Experimental:    true,
		IgnitionVersion: types.MaxVersion,
	}
}

func (specTranslator) Parse(input []byte) (interface{}, error) {
	parsed := &parsedConfig{}
	contextTree, err := cutil.Unmarshal(input, &parsed.config)
	if err != nil {
		return nil, err
	}
	parsed.contextTree = contextTree
	return parsed, nil
}

func (specTranslator) Validate(input interface{}) (report.Report, error) {
	parsed, err := getParsedConfig(input)
	if err != nil {
		return report.Report{}, err
	}
	return cutil.ValidateSourceConfig(&parsed.config, parsed.contextTree)
}

func (specTranslator) Translate(input interface{}, options translator.Options) (interface{}, report.Report, error) {
	parsed, err := getParsedConfig(input)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	translateOptions := common.TranslateOptions{
		FilesDir:                  options.FilesDir,
		NoResourceAutoCompression: options.NoResourceAutoCompression,
		DebugPrintTranslations:    options.DebugPrintTranslations,
	}
	final, translations, translationReport := parsed.config.ToIgn3_7Unvalidated(translateOptions)
	r, err := cutil.ValidateTranslatedConfig(parsed.config, final, translations, translationReport, translateOptions)
	r.Correlate(parsed.contextTree)
	if err != nil {
		return types.Config{}, r, err
	}
	return final, r, nil
}

func getParsedConfig(input interface{}) (*parsedConfig, error) {
	parsed, ok := input.(*parsedConfig)
	if !ok || parsed == nil {
		return nil, fmt.Errorf("fcos v1.8 experimental translator: unexpected parsed config type %T", input)
	}
	return parsed, nil
}
