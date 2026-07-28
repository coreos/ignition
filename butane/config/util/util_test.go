// Copyright 2019 Red Hat, Inc
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

	"github.com/coreos/butane/config/common"
	"github.com/coreos/butane/translate"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

func TestSnake(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{},
		{
			"foo",
			"foo",
		},
		{
			"snakeCase",
			"snake_case",
		},
		{
			"longSnakeCase",
			"long_snake_case",
		},
		{
			"snake_already",
			"snake_already",
		},
		{
			"camelMiB",
			"camel_mib",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if Snake(test.in) != test.out {
				t.Errorf("expected %q got %q", test.out, Snake(test.in))
			}
		})
	}
}

func TestCamel(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{},
		{
			"foo",
			"foo",
		},
		{
			"snake_case",
			"snakeCase",
		},
		{
			"long_snake_case",
			"longSnakeCase",
		},
		{
			"camelAlready",
			"camelAlready",
		},
		{
			"snake_mib",
			"snakeMiB",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if Camel(test.in) != test.out {
				t.Errorf("expected %q got %q", test.out, Camel(test.in))
			}
		})
	}
}

func TestTranslateReportPaths(t *testing.T) {
	ts := translate.NewTranslationSet("yaml", "json")
	ts.AddTranslation(path.New("yaml", "a", "b", "c"), path.New("json", "d", "e", "f"))
	makeReport := func(source bool) report.Report {
		var r report.Report
		var p path.ContextPath
		if source {
			p = path.New("yaml", "a", "b", "c")
		} else {
			p = path.New("json", "d", "e", "f")
		}
		r.AddOnError(p, common.ErrDecimalMode)
		return r
	}
	r := makeReport(false)
	r2 := TranslateReportPaths(r, ts)
	assert.Equal(t, makeReport(false), r, "TranslateReportPaths changed original report")
	assert.Equal(t, makeReport(true), r2, "TranslateReportPaths returned incorrect report")
}
