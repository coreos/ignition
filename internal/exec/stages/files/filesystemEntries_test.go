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
	"os"
	"path/filepath"
	"testing"
)

// TestIsOstreeSystem tests the ostree system detection function
func TestIsOstreeSystem(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		expected    bool
		description string
	}{
		{
			name: "ostree system",
			setupFunc: func(tmpDir string) error {
				ostreeMarker := filepath.Join(tmpDir, "run", "ostree-booted")
				if err := os.MkdirAll(filepath.Dir(ostreeMarker), 0755); err != nil {
					return err
				}
				return os.WriteFile(ostreeMarker, []byte(""), 0644)
			},
			expected:    true,
			description: "Should detect ostree system when /run/ostree-booted exists",
		},
		{
			name: "non-ostree system",
			setupFunc: func(tmpDir string) error {
				// Just create the run directory without the ostree-booted file
				return os.MkdirAll(filepath.Join(tmpDir, "run"), 0755)
			},
			expected:    false,
			description: "Should not detect ostree when /run/ostree-booted is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "ignition-test-")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					t.Logf("failed to remove temp dir: %v", err)
				}
			}()

			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			result := isOstreeSystem(filepath.Join(tmpDir, "run", "ostree-booted"))
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.expected)
			}
		})
	}
}

// TestCrypttabOptionsGeneration tests that the crypttab options are generated correctly
func TestCrypttabOptionsGeneration(t *testing.T) {
	tests := []struct {
		name           string
		hasNetworkDev  bool
		isOstree       bool
		expectedResult string
		expectedFormat string
		description    string
	}{
		{
			name:           "regular system without network",
			hasNetworkDev:  false,
			isOstree:       false,
			expectedResult: "",
			expectedFormat: "testdev UUID=test-uuid /path/to/keyfile luks\n",
			description:    "Should have no options on non-ostree without network",
		},
		{
			name:           "regular system with network",
			hasNetworkDev:  true,
			isOstree:       false,
			expectedResult: ",_netdev",
			expectedFormat: "testdev UUID=test-uuid /path/to/keyfile luks,_netdev\n",
			description:    "Should have _netdev on non-ostree with network",
		},
		{
			name:           "ostree system without network",
			hasNetworkDev:  false,
			isOstree:       true,
			expectedResult: ",x-initrd.attach",
			expectedFormat: "testdev UUID=test-uuid /path/to/keyfile luks,x-initrd.attach\n",
			description:    "Should have x-initrd.attach on ostree without network",
		},
		{
			name:           "ostree system with network",
			hasNetworkDev:  true,
			isOstree:       true,
			expectedResult: ",_netdev,x-initrd.attach",
			expectedFormat: "testdev UUID=test-uuid /path/to/keyfile luks,_netdev,x-initrd.attach\n",
			description:    "Should have both _netdev and x-initrd.attach on ostree with network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildCrypttabOptions(tt.hasNetworkDev, tt.isOstree)

			// Check the options string is exactly what we expect
			if options != tt.expectedResult {
				t.Errorf("%s:\n  got options:  %q\n  want options: %q", tt.description, options, tt.expectedResult)
			}

			// Build the full crypttab line for format verification
			crypttabLine := "testdev UUID=test-uuid /path/to/keyfile luks" + options + "\n"
			if crypttabLine != tt.expectedFormat {
				t.Errorf("%s:\n  got line:  %q\n  want line: %q", tt.description, crypttabLine, tt.expectedFormat)
			}
		})
	}
}

// TestCrypttabOptionsWithClevis tests crypttab options when using Clevis
func TestCrypttabOptionsWithClevis(t *testing.T) {
	tests := []struct {
		name            string
		hasNetworkDev   bool
		isOstree        bool
		expectedKeyfile string
		expectedOptions string
		expectedLine    string
		description     string
	}{
		{
			name:            "clevis on regular system",
			hasNetworkDev:   true,
			isOstree:        false,
			expectedKeyfile: "none",
			expectedOptions: ",_netdev",
			expectedLine:    "testdev UUID=test-uuid none luks,_netdev\n",
			description:     "Clevis should use 'none' as keyfile on non-ostree",
		},
		{
			name:            "clevis on ostree system",
			hasNetworkDev:   true,
			isOstree:        true,
			expectedKeyfile: "none",
			expectedOptions: ",_netdev,x-initrd.attach",
			expectedLine:    "testdev UUID=test-uuid none luks,_netdev,x-initrd.attach\n",
			description:     "Clevis should use 'none' as keyfile and have x-initrd.attach on ostree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildCrypttabOptions(tt.hasNetworkDev, tt.isOstree)

			// Check the options are correct
			if options != tt.expectedOptions {
				t.Errorf("%s:\n  got options:  %q\n  want options: %q", tt.description, options, tt.expectedOptions)
			}

			// Verify full crypttab entry format
			crypttabLine := "testdev UUID=test-uuid " + tt.expectedKeyfile + " luks" + options + "\n"
			if crypttabLine != tt.expectedLine {
				t.Errorf("%s:\n  got line:  %q\n  want line: %q", tt.description, crypttabLine, tt.expectedLine)
			}
		})
	}
}
