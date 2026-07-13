// Copyright 2019 CoreOS, Inc.
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
	butaneconfig "github.com/coreos/ignition/v2/butane/config"
	"github.com/coreos/ignition/v2/butane/config/common"
	exp "github.com/coreos/ignition/v2/config/v3_7_experimental"
	types_exp "github.com/coreos/ignition/v2/config/v3_7_experimental/types"

	"github.com/coreos/vcontext/report"
)

// Parse parses a config of any supported version and returns the
// equivalent config at the latest supported version. It first attempts
// to parse the input as Ignition JSON. If that fails, it attempts to
// transpile the input as a Butane YAML config, allowing users to
// provide Butane configs directly without a separate transpilation
// step.
func Parse(raw []byte) (types_exp.Config, report.Report, error) {
	// Try standard Ignition JSON first
	cfg, rpt, err := exp.ParseCompatibleVersion(raw)
	if err == nil {
		return cfg, rpt, nil
	}

	// JSON failed -- try Butane YAML transpilation
	ignJSON, butaneRpt, butaneErr := butaneconfig.TranslateBytes(raw, common.TranslateBytesOptions{})
	if butaneErr != nil {
		// Both failed -- return original Ignition error for backward compat
		return types_exp.Config{}, rpt, err
	}

	// Butane succeeded -- parse the resulting Ignition JSON
	cfg, parseRpt, parseErr := exp.ParseCompatibleVersion(ignJSON)
	butaneRpt.Merge(parseRpt)
	return cfg, butaneRpt, parseErr
}
