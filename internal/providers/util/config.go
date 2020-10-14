// Copyright 2016 CoreOS, Inc.
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

package util

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/coreos/ignition/v2/config"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"

	"github.com/coreos/vcontext/report"
)

func ParseConfig(logger *log.Logger, rawConfig []byte) (types.Config, report.Report, error) {
	hash := sha512.Sum512(rawConfig)
	logger.Debug("parsing config with SHA512: %s", hex.EncodeToString(hash[:]))

	return config.Parse(rawConfig)
}
