// Copyright 2026 Red Hat, Inc
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

package main

import (
	"reflect"
	"testing"
)

func TestParseFileProcessorOptions(t *testing.T) {
	for _, test := range []struct {
		name, processor, extensions string
		processorSet, extensionsSet bool
		want                        []string
		wantError                   bool
	}{
		{name: "disabled"},
		{name: "missing allowlist", processor: "renderer", processorSet: true, wantError: true},
		{name: "empty allowlist", processor: "renderer", processorSet: true, extensionsSet: true, wantError: true},
		{name: "blank allowlist", processor: "renderer", extensions: "  ", processorSet: true, extensionsSet: true, wantError: true},
		{name: "missing processor", extensions: "bu", extensionsSet: true, wantError: true},
		{name: "empty processor", extensions: "bu", processorSet: true, extensionsSet: true, wantError: true},
		{name: "explicit empty processor alone", processorSet: true, wantError: true},
		{name: "explicit empty allowlist alone", extensionsSet: true, wantError: true},
		{name: "empty entry", processor: "renderer", extensions: "bu,,txt", processorSet: true, extensionsSet: true, wantError: true},
		{name: "dot entry", processor: "renderer", extensions: ".", processorSet: true, extensionsSet: true, wantError: true},
		{name: "normalized suffixes", processor: "renderer", extensions: " .bu, templated.yml,stdin,* ", processorSet: true, extensionsSet: true, want: []string{"bu", "templated.yml", "stdin", "*"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFileProcessorOptions(test.processor, test.extensions, test.processorSet, test.extensionsSet)
			if (err != nil) != test.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestMatchesExtension(t *testing.T) {
	for _, test := range []struct {
		name       string
		extensions []string
		want       bool
	}{
		{name: "config.bu"},
		{name: "config.bu", extensions: []string{"bu"}, want: true},
		{name: "config.templated.yml", extensions: []string{"templated.yml"}, want: true},
		{name: "config.yml", extensions: []string{"templated.yml"}},
		{name: "config.raw", extensions: []string{"bu", "txt"}},
		{name: "nested/config.txt", extensions: []string{"bu", "txt"}, want: true},
		{name: "binary", extensions: []string{"*"}, want: true},
		{name: "config.BU", extensions: []string{"bu"}},
		{name: "parent.bu/config.txt", extensions: []string{"bu"}},
	} {
		if got := matchesExtension(test.name, test.extensions); got != test.want {
			t.Errorf("matchesExtension(%q, %v) = %v, want %v", test.name, test.extensions, got, test.want)
		}
	}
}
