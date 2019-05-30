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
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_0"
	types_3_0 "github.com/coreos/ignition/v2/config/v3_0/types"
	"github.com/coreos/ignition/v2/config/v3_1_experimental"
	trans_exp "github.com/coreos/ignition/v2/config/v3_1_experimental/translate"
	types_exp "github.com/coreos/ignition/v2/config/v3_1_experimental/types"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/vcontext/report"
)

type versionStub struct {
	Ignition struct {
		Version string
	}
}

// Parse parses a config of any supported version and returns the equivalent config at the latest
// supported version.
func Parse(raw []byte) (types_exp.Config, report.Report, error) {
	if len(raw) == 0 {
		return types_exp.Config{}, report.Report{}, errors.ErrEmpty
	}

	stub := versionStub{}
	rpt, err := util.HandleParseErrors(raw, &stub)
	if err != nil {
		return types_exp.Config{}, rpt, err
	}

	version, err := semver.NewVersion(stub.Ignition.Version)
	if err != nil {
		return types_exp.Config{}, report.Report{}, errors.ErrInvalidVersion
	}

	switch *version {
	case types_exp.MaxVersion:
		return v3_1_experimental.Parse(raw)
	case types_3_0.MaxVersion:
		return from3_0(v3_0.Parse(raw))
	default:
		return types_exp.Config{}, report.Report{}, errors.ErrUnknownVersion
	}
}

func from3_0(cfg types_3_0.Config, r report.Report, err error) (types_exp.Config, report.Report, error) {
	if err != nil {
		return types_exp.Config{}, r, err
	}
	return trans_exp.Translate(cfg), r, nil
}
