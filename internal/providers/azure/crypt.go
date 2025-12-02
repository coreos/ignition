// Copyright 2015 CoreOS, Inc.
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

package azure

import (
	"strings"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt" // Register SHA-512 crypt
)

// HashPassword hashes a plaintext password using SHA-512 crypt algorithm.
// Returns a string suitable for /etc/shadow in the format $6$<salt>$<hash>
func HashPassword(password string) (string, error) {
	crypter := crypt.SHA512.New()
	return crypter.Generate([]byte(password), nil)
}

// IsPasswordHashed checks if a password string appears to already be hashed
// (starts with a known crypt prefix like $6$, $5$, $y$, etc.)
func IsPasswordHashed(password string) bool {
	hashPrefixes := []string{
		"$6$", // SHA-512
		"$5$", // SHA-256
		"$y$", // yescrypt
		"$2a$", "$2b$", "$2y$", // bcrypt variants
		"$1$", // MD5 (deprecated)
	}
	for _, prefix := range hashPrefixes {
		if strings.HasPrefix(password, prefix) {
			return true
		}
	}
	return false
}
