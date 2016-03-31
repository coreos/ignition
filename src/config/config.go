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

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/v1"

	"github.com/coreos/ignition/third_party/github.com/camlistore/camlistore/pkg/errorutil"
)

var (
	ErrCloudConfig = errors.New("not a config (found coreos-cloudconfig)")
	ErrEmpty       = errors.New("not a config (empty)")
	ErrScript      = errors.New("not a config (found coreos-cloudinit script)")
	ErrDeprecated  = errors.New("config format deprecated")
)

func Parse(rawConfig []byte) (types.Config, error) {
	major, err := majorVersion(rawConfig)
	if err != nil {
		return types.Config{}, err
	}

	switch major {
	case 1:
		config, err := ParseFromV1(rawConfig)
		if err != nil {
			return types.Config{}, err
		}

		return config, ErrDeprecated
	default:
		return ParseFromLatest(rawConfig)
	}
}

func ParseFromLatest(rawConfig []byte) (config types.Config, err error) {
	if err = json.Unmarshal(rawConfig, &config); err == nil {
		err = config.Ignition.Version.AssertValid()
	} else if isEmpty(rawConfig) {
		err = ErrEmpty
	} else if isCloudConfig(rawConfig) {
		err = ErrCloudConfig
	} else if isScript(rawConfig) {
		err = ErrScript
	}
	if serr, ok := err.(*json.SyntaxError); ok {
		line, col, highlight := errorutil.HighlightBytePosition(bytes.NewReader(rawConfig), serr.Offset)
		err = fmt.Errorf("error at line %d, column %d\n%s%v", line, col, highlight, err)
	}

	return
}

func ParseFromV1(rawConfig []byte) (types.Config, error) {
	config, err := v1.Parse(rawConfig)
	if err != nil {
		return types.Config{}, err
	}

	return TranslateFromV1(config)
}

type config struct {
	Version  *int `json:"ignitionVersion" yaml:"ignition_version"`
	Ignition struct {
		Version *types.IgnitionVersion `json:"version,omitempty" yaml:"version" merge:"old"`
	} `json:"ignition" yaml:"ignition"`
}

func majorVersion(rawConfig []byte) (int64, error) {
	var composite config
	if err := json.Unmarshal(rawConfig, &composite); err != nil {
		if serr, ok := err.(*json.SyntaxError); ok {
			line, col, highlight := errorutil.HighlightBytePosition(bytes.NewReader(rawConfig), serr.Offset)
			err = fmt.Errorf("error at line %d, column %d\n%s%v", line, col, highlight, err)
		}
		return 0, err
	}

	var major int64
	if composite.Ignition.Version != nil {
		major = composite.Ignition.Version.Major
	} else if composite.Version != nil {
		major = int64(*composite.Version)
	}

	return major, nil
}

func isEmpty(userdata []byte) bool {
	return len(userdata) == 0
}
