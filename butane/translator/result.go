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

import (
	"github.com/coreos/butane/translate"
	"github.com/coreos/vcontext/report"
)

// Result contains the output of a translation operation.
//
// This matches the existing return pattern from ToIgnXXBytes functions
// but wraps them in a struct for better extensibility.
type Result struct {
	// Output is the translated Ignition configuration as JSON bytes.
	Output []byte

	// Report contains warnings and errors from the translation process.
	// Use Report.IsFatal() to check if translation failed.
	Report report.Report

	// TranslationSet tracks how source paths in the Butane config map to
	// output paths in the Ignition config. Used for debugging and tooling.
	TranslationSet translate.TranslationSet
}
