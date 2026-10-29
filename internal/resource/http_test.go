// Copyright 2026 Red Hat, Inc.
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

package resource

import "testing"

func TestShouldRetryHttp(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		retryCodes []int
		out        bool
	}{
		{"nil", 429, nil, false},
		{"empty", 429, []int{}, false},
		{"one", 429, []int{429}, true},
		{"first", 408, []int{408, 429}, true},
		{"last", 429, []int{408, 429}, true},
		{"missing", 404, []int{408, 429}, false},
		{"below server error", 499, nil, false},
		{"server error", 500, nil, true},
		{"higher server error", 503, []int{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := FetchOptions{RetryCodes: test.retryCodes}
			if got := shouldRetryHttp(test.statusCode, opts); got != test.out {
				t.Errorf("shouldRetryHttp(%d, %+v) = %t, want %t", test.statusCode, opts, got, test.out)
			}
		})
	}
}
