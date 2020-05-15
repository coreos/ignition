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
	converter "github.com/coreos/ign-converter"
	"github.com/coreos/ignition/config/shared/errors"
	currentExperimentalv2 "github.com/coreos/ignition/config/v2_5_experimental"
	"github.com/coreos/ignition/config/validate/report"
	"github.com/coreos/ignition/internal/config/types"
	v3 "github.com/coreos/ignition/v2/config/v3_2_experimental"
)

func Parse(rawConfig []byte) (types.Config, report.Report, error) {
	cfg, rpt, err := currentExperimentalv2.Parse(rawConfig)
	if err == nil && !rpt.IsFatal() {
		return Translate(cfg), rpt, nil
	}
	if err.Error() == errors.ErrUnknownVersion.Error() {
		// compat mode for converting spec v3 config
		cfgv3, rptv3, err := v3.Parse(rawConfig)
		if err != nil {
			return types.Config{}, report.Merge(rpt, report.Add(report.Entry{rptv3.String()})), err
		}
		cfgv2, err := converter.Translate3to2(cfgv3)
		if err != nil {
			return types.Config{}, report.Merge(rpt, report.Add(report.Entry{rptv3.String()})), err
		}

		return Translate(cfgv2), report.Merge(rpt, report.Add(report.Entry{rptv3.String()})), nil
	}

	return types.Config{}, rpt, err
}
