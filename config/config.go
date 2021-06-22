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
	exp "github.com/coreos/ignition/v2/config/v3_4_experimental"
	types_exp "github.com/coreos/ignition/v2/config/v3_4_experimental/types"

	"github.com/coreos/vcontext/report"
)

// Parse parses a config of any supported version and returns the equivalent config at the latest
// supported version.
func Parse(raw []byte) (types_exp.Config, report.Report, error) {
	return exp.ParseCompatibleVersion(raw)
}
