// Copyright 2026 CoreOS, Inc.
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

package files

import (
	"testing"
)

func TestBuildCrypttabOptions(t *testing.T) {
	tests := []struct {
		name           string
		hasNetworkDev  bool
		expectedResult string
		expectedFormat string
	}{
		{
			name:           "without network",
			hasNetworkDev:  false,
			expectedResult: ",x-initrd.attach",
			expectedFormat: "testdev UUID=test-uuid /path/to/keyfile luks,x-initrd.attach\n",
		},
		{
			name:           "with network",
			hasNetworkDev:  true,
			expectedResult: ",_netdev,x-initrd.attach",
			expectedFormat: "testdev UUID=test-uuid /path/to/keyfile luks,_netdev,x-initrd.attach\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildCrypttabOptions(tt.hasNetworkDev)

			if options != tt.expectedResult {
				t.Errorf("got options %q, want %q", options, tt.expectedResult)
			}

			crypttabLine := "testdev UUID=test-uuid /path/to/keyfile luks" + options + "\n"
			if crypttabLine != tt.expectedFormat {
				t.Errorf("got line %q, want %q", crypttabLine, tt.expectedFormat)
			}
		})
	}
}

func TestBuildCrypttabOptionsWithClevis(t *testing.T) {
	tests := []struct {
		name            string
		hasNetworkDev   bool
		expectedOptions string
		expectedLine    string
	}{
		{
			name:            "clevis without network",
			hasNetworkDev:   false,
			expectedOptions: ",x-initrd.attach",
			expectedLine:    "testdev UUID=test-uuid none luks,x-initrd.attach\n",
		},
		{
			name:            "clevis with network",
			hasNetworkDev:   true,
			expectedOptions: ",_netdev,x-initrd.attach",
			expectedLine:    "testdev UUID=test-uuid none luks,_netdev,x-initrd.attach\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildCrypttabOptions(tt.hasNetworkDev)

			if options != tt.expectedOptions {
				t.Errorf("got options %q, want %q", options, tt.expectedOptions)
			}

			crypttabLine := "testdev UUID=test-uuid none luks" + options + "\n"
			if crypttabLine != tt.expectedLine {
				t.Errorf("got line %q, want %q", crypttabLine, tt.expectedLine)
			}
		})
	}
}
