// Copyright 2010 The Go Authors.
// Copyright 2018 The Ignition Authors.
// All rights reserved.
// Use of this source code is governed by a BSD-style license.

// Package earlyrand implements a early-boot cryptographically
// mostly-secure random number generator.
package earlyrand

import "io"

// Reader is a global, shared instance of the random number generator.
//
// Reader uses non-blocking getrandom(2) if possible, /dev/urandom otherwise.
var Reader io.Reader

// Read is a helper function that calls Reader.Read using io.ReadFull.
// On return, n == len(b) if and only if err == nil.
func Read(b []byte) (n int, err error) {
	return io.ReadFull(Reader, b)
}
