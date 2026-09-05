// Copyright 2022 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import "github.com/coreos/vcontext/report"

// Translator translates Butane configuration to Ignition configuration.
//
// Each Butane variant (fcos, flatcar, r4e, openshift, etc.) should implement this
// interface for each supported version.
type Translator interface {
	// Metadata returns the variant, version, and target Ignition version.
	Metadata() Metadata
	// Parse parses YAML into the translator's schema type.
	Parse(input []byte) (interface{}, error)
	// Translate translates a parsed config after successful validation.
	Translate(input interface{}, options Options) (interface{}, report.Report, error)
	// Validate validates a parsed config.
	Validate(in interface{}) (report.Report, error)
}
