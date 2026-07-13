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

package util

import (
	"fmt"
	"testing"

	"github.com/coreos/butane/translate"

	"github.com/coreos/ignition/v2/config/util"
	// config version doesn't matter; just pick one
	"github.com/coreos/ignition/v2/config/v3_0/types"
	"github.com/coreos/vcontext/path"
	"github.com/stretchr/testify/assert"
)

// TestMergeTranslatedConfigs tests merging two Ignition configs and their
// corresponding translations.
func TestMergeTranslatedConfigs(t *testing.T) {
	tests := []struct {
		parent             types.Config
		parentTranslations translate.TranslationSet
		child              types.Config
		childTranslations  translate.TranslationSet
		merged             types.Config
		mergedTranslations translate.TranslationSet
	}{
		{
			parent: types.Config{
				Ignition: types.Ignition{
					Version: "3.0.0",
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Name:     "aardvark.service",
							Enabled:  util.BoolToPtr(true),
							Contents: util.StrToPtr("antelope"),
						},
						{
							Name:     "caribou.service",
							Contents: util.StrToPtr("caribou"),
						},
						{
							Name:     "elephant.service",
							Contents: util.StrToPtr("elephant"),
						},
					},
				},
			},
			parentTranslations: makeTranslationSet([]translate.Translation{
				// parent key duplicated in child, should be clobbered
				{From: path.New("in", "bad", 1), To: path.New("out", "systemd", "units", 0, "name")},
				// parent field overridden in child, should be clobbered
				{From: path.New("in", "bad", 2), To: path.New("out", "systemd", "units", 0, "contents")},
				// parent field not overridden in child
				{From: path.New("in", "good", 1), To: path.New("out", "systemd", "units", 0, "enabled")},
				// parent key not specified in child
				{From: path.New("in", "good", 2), To: path.New("out", "systemd", "units", 1, "name")},
				// parent field not specified in child
				{From: path.New("in", "good", 3), To: path.New("out", "systemd", "units", 1, "contents")},
				// other fields omitted from translation set
			}),
			child: types.Config{
				Ignition: types.Ignition{
					Version: "3.0.0",
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Name:     "bear.service",
							Enabled:  util.BoolToPtr(true),
							Contents: util.StrToPtr("bear"),
						},
						{
							Name:     "aardvark.service",
							Contents: util.StrToPtr("aardvark"),
						},
					},
				},
			},
			childTranslations: makeTranslationSet([]translate.Translation{
				// child key not mentioned in parent
				{From: path.New("in", "good", 11), To: path.New("out", "systemd", "units", 0, "name")},
				// child field not mentioned in parent
				{From: path.New("in", "good", 12), To: path.New("out", "systemd", "units", 0, "contents")},
				// parent key duplicated in child
				{From: path.New("in", "good", 13), To: path.New("out", "systemd", "units", 1, "name")},
				// parent field overridden in child
				{From: path.New("in", "good", 14), To: path.New("out", "systemd", "units", 1, "contents")},
				// other fields omitted from translation set
			}),
			merged: types.Config{
				Ignition: types.Ignition{
					Version: "3.0.0",
				},
				Systemd: types.Systemd{
					Units: []types.Unit{
						{
							Name:     "aardvark.service",
							Enabled:  util.BoolToPtr(true),
							Contents: util.StrToPtr("aardvark"),
						},
						{
							Name:     "caribou.service",
							Contents: util.StrToPtr("caribou"),
						},
						{
							Name:     "elephant.service",
							Contents: util.StrToPtr("elephant"),
						},
						{
							Name:     "bear.service",
							Enabled:  util.BoolToPtr(true),
							Contents: util.StrToPtr("bear"),
						},
					},
				},
			},
			mergedTranslations: makeTranslationSet([]translate.Translation{
				{From: path.New("in", "good", 13), To: path.New("out", "systemd", "units", 0, "name")},
				{From: path.New("in", "good", 1), To: path.New("out", "systemd", "units", 0, "enabled")},
				{From: path.New("in", "good", 14), To: path.New("out", "systemd", "units", 0, "contents")},
				{From: path.New("in", "good", 2), To: path.New("out", "systemd", "units", 1, "name")},
				{From: path.New("in", "good", 3), To: path.New("out", "systemd", "units", 1, "contents")},
				{From: path.New("in", "good", 11), To: path.New("out", "systemd", "units", 3, "name")},
				{From: path.New("in", "good", 12), To: path.New("out", "systemd", "units", 3, "contents")},
			}),
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("merge %d", i), func(t *testing.T) {
			c, ts := MergeTranslatedConfigs(test.parent, test.parentTranslations, test.child, test.childTranslations)
			assert.Equal(t, test.merged, c, "bad config")
			assert.Equal(t, test.mergedTranslations, ts, "bad translations")
		})
	}
}

func makeTranslationSet(translations []translate.Translation) translate.TranslationSet {
	ts := translate.NewTranslationSet(translations[0].From.Tag, translations[0].To.Tag)
	for _, t := range translations {
		ts.AddTranslation(t.From, t.To)
	}
	return ts
}
