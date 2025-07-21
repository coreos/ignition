// Copyright 2024 Solar Designer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Alternatively, this specific source file is also available under more
// relaxed terms (0-clause BSD license):
// Redistribution and use in source and binary forms, with or without
// modification, are permitted.

// yescrypt support sponsored by Sandfly Security https://sandflysecurity.com -
// Agentless Security for Linux

package yescrypt

import (
	"bytes"
	"errors"
)

const itoa64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var atoi64Partial = [...]byte{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	64, 64, 64, 64, 64, 64, 64,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24,
	25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37,
	64, 64, 64, 64, 64, 64,
	38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
}

func atoi64(c byte) int {
	if c >= '.' && c <= 'z' {
		return int(atoi64Partial[c-'.'])
	}
	return 64
}

func encode64(src []byte) []byte {
	dst := make([]byte, 0, (len(src)*8+5)/6)
	for i := 0; i < len(src); {
		value, bits := uint32(0), 0
		for ; bits < 24 && i < len(src); bits += 8 {
			value |= uint32(src[i]) << bits
			i++
		}
		for ; bits > 0; bits -= 6 {
			dst = append(dst, itoa64[value&0x3f])
			value >>= 6
		}
	}
	return dst
}

func decode64(src []byte) []byte {
	dst := make([]byte, 0, len(src)*3/4)
	for i := 0; i < len(src); {
		value, bits := uint32(0), uint32(0)
		for ; bits < 24 && i < len(src); bits += 6 {
			c := atoi64(src[i])
			if c > 63 {
				return nil
			}
			i++
			value |= uint32(c) << bits
		}
		if bits < 12 { // Must have at least one full byte
			return nil
		}
		for ; bits >= 8; bits -= 8 {
			dst = append(dst, byte(value))
			value >>= 8
		}
		if value != 0 { // May have 2 or 4 bits left, which must be 0
			return nil
		}
	}
	return dst
}

// Computes yescrypt hash encoding given the password and existing yescrypt
// setting or full hash encoding.  The salt and other parameters are decoded
// from setting.  Currently supports (only a little more than) the subset of
// yescrypt parameters that libxcrypt can generate (as of libxcrypt 4.4.36).
func Hash(password, setting []byte) ([]byte, error) {
	if len(setting) < 7 || string(setting[:4]) != "$y$j" || setting[6] != '$' {
		return nil, errors.New("yescrypt: unsupported parameters")
	}
	// Proper yescrypt uses variable-length integers
	// We take a shortcut approach that works in a more limited range
	Nlog2 := atoi64(setting[4]) + 1
	if Nlog2 < 10 || Nlog2 > 18 {
		return nil, errors.New("yescrypt: N out of supported range")
	}
	r := atoi64(setting[5]) + 1
	if r < 1 || r > 32 {
		return nil, errors.New("yescrypt: r out of supported range")
	}

	saltEnd := bytes.LastIndexByte(setting, '$')
	if saltEnd < 7 {
		saltEnd = len(setting)
	}
	salt := decode64(setting[7:saltEnd])
	if salt == nil {
		return nil, errors.New("yescrypt: bad salt encoding")
	}

	key, err := Key(password, salt, 1<<Nlog2, r, 1, 32)
	if err != nil {
		return nil, err
	}

	hash := encode64(key)

	return bytes.Join([][]byte{setting[0:saltEnd], hash}, []byte("$")), nil
}
