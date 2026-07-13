// Copyright 2021 Red Hat, Inc
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

package v1_0

import (
	"fmt"
	"testing"

	baseutil "github.com/coreos/butane/base/util"
	base "github.com/coreos/butane/base/v0_4"
	"github.com/coreos/butane/config/common"
	confutil "github.com/coreos/butane/config/util"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

// Test translation of Flatcar support for Ignition config fields.
func TestTranslation(t *testing.T) {
	type entry struct {
		kind report.EntryKind
		err  error
		path path.ContextPath
	}
	tests := []struct {
		in      Config
		entries []entry
	}{
		// all the warnings/errors
		{
			Config{
				Config: base.Config{
					Storage: base.Storage{
						Luks: []base.Luks{
							{
								Name:   "data",
								Device: util.StrToPtr("/dev/disk/by-partlabel/USR-B"),
							},
							{
								Name:   "data-bis",
								Device: util.StrToPtr("/dev/disk/by-partlabel/USR-B-bis"),
								Clevis: base.Clevis{Tpm2: util.BoolToPtr(true)},
							},
						},
					},
				},
			},
			[]entry{
				{report.Error, common.ErrClevisSupport, path.New("yaml", "storage", "luks", 1, "clevis")},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("translate %d", i), func(t *testing.T) {
			var expectedReport report.Report
			for _, entry := range test.entries {
				expectedReport.AddOn(entry.path, entry.err, entry.kind)
			}
			actual, translations, r := test.in.ToIgn3_3Unvalidated(common.TranslateOptions{})
			r.Merge(fieldFilters.Verify(actual))
			r = confutil.TranslateReportPaths(r, translations)
			baseutil.VerifyReport(t, test.in, r)
			assert.Equal(t, expectedReport, r, "report mismatch")
			assert.NoError(t, translations.DebugVerifyCoverage(actual), "incomplete TranslationSet coverage")
		})
	}
}
