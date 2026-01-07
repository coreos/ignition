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
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/GehirnInc/crypt/sha512_crypt"
)

// HashPassword hashes a plaintext password using SHA-512 crypt.
// Returns a string in the format $6$<salt>$<hash>.
func HashPassword(password string) (string, error) {
	salt, err := generateSalt(16)
	if err != nil {
		return "", err
	}

	crypt := sha512_crypt.New()
	return crypt.Generate([]byte(password), []byte("$6$"+salt))
}

// IsPasswordHashed checks if a password string is already hashed.
// It recognizes common hash formats: SHA-512 ($6$), SHA-256 ($5$),
// yescrypt ($y$), bcrypt ($2a$, $2b$, $2y$), and MD5 ($1$).
func IsPasswordHashed(password string) bool {
	if password == "" {
		return false
	}

	// Check for common password hash prefixes
	hashPrefixes := []string{
		"$6$",  // SHA-512
		"$5$",  // SHA-256
		"$y$",  // yescrypt
		"$2a$", // bcrypt
		"$2b$", // bcrypt
		"$2y$", // bcrypt
		"$1$",  // MD5
	}

	for _, prefix := range hashPrefixes {
		if strings.HasPrefix(password, prefix) {
			// Verify it has the expected structure (at least prefix + something)
			remaining := password[len(prefix):]
			if len(remaining) > 0 && strings.Contains(remaining, "$") {
				return true
			}
		}
	}

	return false
}

// generateSalt generates a random salt of the specified length.
func generateSalt(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use base64 encoding and trim to desired length
	salt := base64.StdEncoding.EncodeToString(bytes)
	// Remove any characters that might cause issues in crypt salt
	salt = strings.ReplaceAll(salt, "+", ".")
	salt = strings.ReplaceAll(salt, "=", "")
	if len(salt) > length {
		salt = salt[:length]
	}
	return salt, nil
}
