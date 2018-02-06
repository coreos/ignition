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

package v2_2

import (
	"bytes"
	"errors"
	"reflect"

	"github.com/coreos/ignition/config/v2_2/types"
	"github.com/coreos/ignition/config/validate"
	astjson "github.com/coreos/ignition/config/validate/astjson"
	"github.com/coreos/ignition/config/validate/report"

	json "github.com/ajeddeloh/go-json"
	"github.com/coreos/go-semver/semver"
	"go4.org/errorutil"
)

var (
	ErrCloudConfig           = errors.New("not a config (found coreos-cloudconfig)")
	ErrEmpty                 = errors.New("not a config (empty)")
	ErrScript                = errors.New("not a config (found coreos-cloudinit script)")
	ErrDeprecated            = errors.New("config format deprecated")
	ErrInvalid               = errors.New("config is not valid")
	ErrUnknownVersion        = errors.New("unsupported config version")
	ErrVersionIndeterminable = errors.New("unable to determine version")
	ErrBadVersion            = errors.New("config must be of version 2.2.0")
)

// Parse parses the raw config into a types.Config struct and generates a report of any
// errors, warnings, info, and deprecations it encountered. Unlike config.Parse,
// it does not attempt to translate the config.
func Parse(rawConfig []byte) (types.Config, report.Report, error) {
	if isEmpty(rawConfig) {
		return types.Config{}, report.Report{}, ErrEmpty
	} else if isCloudConfig(rawConfig) {
		return types.Config{}, report.Report{}, ErrCloudConfig
	} else if isScript(rawConfig) {
		return types.Config{}, report.Report{}, ErrScript
	}

	version, err := Version(rawConfig)
	if err != nil {
		return types.Config{}, report.ReportFromError(err, report.EntryError), err
	}
	if (version != semver.Version{Major: 2, Minor: 2}) {
		return types.Config{}, report.ReportFromError(ErrBadVersion, report.EntryError), ErrBadVersion
	}

	var config types.Config

	// These errors are fatal and the config should not be further validated
	if err = json.Unmarshal(rawConfig, &config); err == nil {
		versionReport := config.Ignition.Validate()
		if versionReport.IsFatal() {
			return types.Config{}, versionReport, ErrInvalid
		}
	}

	// Handle json syntax and type errors first, since they are fatal but have offset info
	if serr, ok := err.(*json.SyntaxError); ok {
		line, col, highlight := errorutil.HighlightBytePosition(bytes.NewReader(rawConfig), serr.Offset)
		return types.Config{},
			report.Report{
				Entries: []report.Entry{{
					Kind:      report.EntryError,
					Message:   serr.Error(),
					Line:      line,
					Column:    col,
					Highlight: highlight,
				}},
			},
			ErrInvalid
	}

	if terr, ok := err.(*json.UnmarshalTypeError); ok {
		line, col, highlight := errorutil.HighlightBytePosition(bytes.NewReader(rawConfig), terr.Offset)
		return types.Config{},
			report.Report{
				Entries: []report.Entry{{
					Kind:      report.EntryError,
					Message:   terr.Error(),
					Line:      line,
					Column:    col,
					Highlight: highlight,
				}},
			},
			ErrInvalid
	}

	// Handle other fatal errors (i.e. invalid version)
	if err != nil {
		return types.Config{}, report.ReportFromError(err, report.EntryError), err
	}

	// Unmarshal again to a json.Node to get offset information for building a report
	var ast json.Node
	var r report.Report
	configValue := reflect.ValueOf(config)
	if err := json.Unmarshal(rawConfig, &ast); err != nil {
		r.Add(report.Entry{
			Kind:    report.EntryWarning,
			Message: "Ignition could not unmarshal your config for reporting line numbers. This should never happen. Please file a bug.",
		})
		r.Merge(validate.ValidateWithoutSource(configValue))
	} else {
		r.Merge(validate.Validate(configValue, astjson.FromJsonRoot(ast), bytes.NewReader(rawConfig), true))
	}

	if r.IsFatal() {
		return types.Config{}, r, ErrInvalid
	}

	return config, r, nil
}

func Version(rawConfig []byte) (semver.Version, error) {
	var composite struct {
		Version  *int `json:"ignitionVersion"`
		Ignition struct {
			Version *string `json:"version"`
		} `json:"ignition"`
	}

	if json.Unmarshal(rawConfig, &composite) == nil {
		if composite.Ignition.Version != nil {
			v, err := types.Ignition{Version: *composite.Ignition.Version}.Semver()
			if err != nil {
				return semver.Version{}, err
			}
			return *v, nil
		} else if composite.Version != nil {
			return semver.Version{Major: int64(*composite.Version)}, nil
		}
	}

	return semver.Version{}, ErrVersionIndeterminable
}

func isEmpty(userdata []byte) bool {
	return len(userdata) == 0
}
