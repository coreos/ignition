// Copyright 2020 Red Hat, Inc
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

package v1_2_exp

import (
	"github.com/coreos/butane/config/common"
	cutil "github.com/coreos/butane/config/util"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/vcontext/report"
)

var (
	fieldFilters = cutil.NewFilters(types.Config{}, cutil.FilterMap{
		"storage.luks.cex": common.ErrCexNotSupported,
	})
)

// Return FieldFilters for this spec.
func (c Config) FieldFilters() *cutil.FieldFilters {
	return &fieldFilters
}

// ToIgn3_5 translates the config to an Ignition config.  It returns a
// report of any errors or warnings in the source and resultant config.  If
// the report has fatal errors or it encounters other problems translating,
// an error is returned.
func (c Config) ToIgn3_6(options common.TranslateOptions) (types.Config, report.Report, error) {
	cfg, r, err := cutil.Translate(c, "ToIgn3_6Unvalidated", options)
	return cfg.(types.Config), r, err
}

// ToIgn3_5Bytes translates from a v1.2 Butane config to a v3.5.0 Ignition config. It returns a report of any errors or
// warnings in the source and resultant config. If the report has fatal errors or it encounters other problems
// translating, an error is returned.
func ToIgn3_6Bytes(input []byte, options common.TranslateBytesOptions) ([]byte, report.Report, error) {
	return cutil.TranslateBytes(input, &Config{}, "ToIgn3_6", options)
}
