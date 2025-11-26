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
	"context"

	"github.com/coreos/vcontext/report"
)

// Translator translates Butane configuration to Ignition configuration.
//
// Each Butane variant (fcos, flatcar, r4e, openshift, etc.) should implement this
// interface for each supported version.
type Translator interface {
	// Metadata the variant, version, and target Ignition version.
	Metadata() Metadata

	// Translate converts Butane config bytes to Ignition config bytes.
	Translate(ctx context.Context, input []byte, opts Options) (Result, error)

	// Validate validates a Butane config without performing translation.
	Validate(ctx context.Context, input []byte) (report.Report, error)
}

