// Copyright 2020 Red Hat, Inc.
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

package types

import (
	"testing"

	"github.com/coreos/vcontext/validate"
)

func TestTLSValidate(t *testing.T) {
	tests := []struct {
		in  TLS
		out string
	}{
		{
			TLS{
				CertificateAuthorities: []Resource{{}},
			},
			"error at $.certificateAuthorities.0: source is required\n",
		},
	}

	for i, test := range tests {
		r := validate.Validate(test.in, "test")
		if test.out != r.String() {
			t.Errorf("#%d: bad error: want %q, got %q", i, test.out, r.String())
		}
	}
}
