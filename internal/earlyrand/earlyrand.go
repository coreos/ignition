// Copyright 2018 - The Ignition authors
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

// Package earlyrand provides a non-blocking reader to a source of randomness
// that can be used at early-boot. If the system-wide CSRNG is not yet initialized,
// this may return non-cryptograhically-secure randomness. This is an explicit
// trade-off that makes this package not suitable for key-generation.
package earlyrand

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// urandomPath is the hardcoded path to a non-blocking source of randomness.
const urandomPath = "/dev/urandom"

// urandomCached caches the file pointer for UrandomReader.
var urandomCached *os.File

// urandomErr caches the error for UrandomReader.
var urandomErr error

// urandomOnce ensures urandomCached is initialized at most once.
var urandomOnce sync.Once

// UrandomReader returns a reader for "/dev/urandom".
func UrandomReader() (io.Reader, error) {
	urandomOnce.Do(func() {
		urandom, err := os.Open(urandomPath)
		if err != nil {
			urandomErr = fmt.Errorf("unable to open %s: %s", urandomPath, err)
			return
		}
		urandomCached = urandom
	})

	return urandomCached, urandomErr
}
