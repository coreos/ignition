// Copyright 2021 Red Hat, Inc.
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

	"github.com/coreos/ignition/v2/config/validate"
)

func TestKernelArgumentsValidate(t *testing.T) {
	tests := []struct {
		in  KernelArguments
		out string
	}{
		// Ensure that ValidateWithContext prevents duplicate entries
		// in ShouldExist & ShouldNotExist
		{
			KernelArguments{
				ShouldExist: []KernelArgument{
					"foo",
					"bar",
				},
				ShouldNotExist: []KernelArgument{
					"baz",
					"foo",
				},
			},
			"error at $.shouldNotExist.1: duplicate entry defined\n",
		},
	}

	for i, test := range tests {
		r := validate.ValidateWithContext(test.in, nil)
		if test.out != r.String() {
			t.Errorf("#%d: bad error: want %q, got %q", i, test.out, r.String())
		}
	}
}
