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
	"github.com/coreos/ignition/config/shared/errors"
	"github.com/coreos/ignition/config/util"
	"github.com/coreos/ignition/config/v3_0_experimental"
	"github.com/coreos/ignition/config/v3_0_experimental/types"
	"github.com/coreos/ignition/config/validate/report"

	"github.com/coreos/go-semver/semver"
)

type versionStub struct {
	Ignition struct {
		Version string
	}
}

// Parse parses a config of any supported version and returns the equivalent config at the latest
// supported version.
func Parse(raw []byte) (types.Config, report.Report, error) {
	if len(raw) == 0 {
		return types.Config{}, report.Report{}, errors.ErrEmpty
	}

	stub := versionStub{}
	rpt, err := util.HandleParseErrors(raw, &stub)
	if err != nil {
		return types.Config{}, rpt, err
	}

	version, err := semver.NewVersion(stub.Ignition.Version)
	if err != nil {
		return types.Config{}, report.Report{}, errors.ErrInvalidVersion
	}

	switch *version {
	case types.MaxVersion:
		return v3_0_experimental.Parse(raw)
	default:
		return types.Config{}, report.Report{}, errors.ErrUnknownVersion
	}
}
