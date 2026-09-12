// Copyright 2012-2020 The Go Authors. All rights reserved.
// Copyright 2024 Solar Designer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yescrypt implements the scrypt key derivation function as defined in
// Colin Percival's paper "Stronger Key Derivation via Sequential Memory-Hard
// Functions", as well as Solar Designer's yescrypt.

// yescrypt support sponsored by Sandfly Security https://sandflysecurity.com -
// Agentless Security for Linux

package yescrypt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/bits"

	"golang.org/x/crypto/pbkdf2"
)

const maxInt = int(^uint(0) >> 1)

// blockCopy copies n numbers from src into dst.
func blockCopy(dst, src []uint64, n int) {
	copy(dst, src[:n])
}

// blockXOR XORs numbers from dst with n numbers from src.
func blockXOR(dst, src []uint64, n int) {
	for i, v := range src[:n] {
		dst[i] ^= v
	}
}

// salsaXOR applies Salsa20/8 to the XOR of 16 numbers from tmp and in,
// and puts the result into both tmp and out.
func salsaXOR(tmp *[8]uint64, in, out []uint64, rounds int) {
	d0 := tmp[0] ^ in[0]
	d1 := tmp[1] ^ in[1]
	d2 := tmp[2] ^ in[2]
	d3 := tmp[3] ^ in[3]
	d4 := tmp[4] ^ in[4]
	d5 := tmp[5] ^ in[5]
	d6 := tmp[6] ^ in[6]
	d7 := tmp[7] ^ in[7]

	x0, x1 := uint32(d0), uint32(d6>>32)
	x2, x3 := uint32(d5), uint32(d3>>32)
	x4, x5 := uint32(d2), uint32(d0>>32)
	x6, x7 := uint32(d7), uint32(d5>>32)
	x8, x9 := uint32(d4), uint32(d2>>32)
	x10, x11 := uint32(d1), uint32(d7>>32)
	x12, x13 := uint32(d6), uint32(d4>>32)
	x14, x15 := uint32(d3), uint32(d1>>32)

	for i := 0; i < rounds; i += 2 {
		x4 ^= bits.RotateLeft32(x0+x12, 7)
		x8 ^= bits.RotateLeft32(x4+x0, 9)
		x12 ^= bits.RotateLeft32(x8+x4, 13)
		x0 ^= bits.RotateLeft32(x12+x8, 18)

		x9 ^= bits.RotateLeft32(x5+x1, 7)
		x13 ^= bits.RotateLeft32(x9+x5, 9)
		x1 ^= bits.RotateLeft32(x13+x9, 13)
		x5 ^= bits.RotateLeft32(x1+x13, 18)

		x14 ^= bits.RotateLeft32(x10+x6, 7)
		x2 ^= bits.RotateLeft32(x14+x10, 9)
		x6 ^= bits.RotateLeft32(x2+x14, 13)
		x10 ^= bits.RotateLeft32(x6+x2, 18)

		x3 ^= bits.RotateLeft32(x15+x11, 7)
		x7 ^= bits.RotateLeft32(x3+x15, 9)
		x11 ^= bits.RotateLeft32(x7+x3, 13)
		x15 ^= bits.RotateLeft32(x11+x7, 18)

		x1 ^= bits.RotateLeft32(x0+x3, 7)
		x2 ^= bits.RotateLeft32(x1+x0, 9)
		x3 ^= bits.RotateLeft32(x2+x1, 13)
		x0 ^= bits.RotateLeft32(x3+x2, 18)

		x6 ^= bits.RotateLeft32(x5+x4, 7)
		x7 ^= bits.RotateLeft32(x6+x5, 9)
		x4 ^= bits.RotateLeft32(x7+x6, 13)
		x5 ^= bits.RotateLeft32(x4+x7, 18)

		x11 ^= bits.RotateLeft32(x10+x9, 7)
		x8 ^= bits.RotateLeft32(x11+x10, 9)
		x9 ^= bits.RotateLeft32(x8+x11, 13)
		x10 ^= bits.RotateLeft32(x9+x8, 18)

		x12 ^= bits.RotateLeft32(x15+x14, 7)
		x13 ^= bits.RotateLeft32(x12+x15, 9)
		x14 ^= bits.RotateLeft32(x13+x12, 13)
		x15 ^= bits.RotateLeft32(x14+x13, 18)
	}

	d0 = uint64(uint32(d0)+x0) | uint64(uint32(d0>>32)+x5)<<32
	d1 = uint64(uint32(d1)+x10) | uint64(uint32(d1>>32)+x15)<<32
	d2 = uint64(uint32(d2)+x4) | uint64(uint32(d2>>32)+x9)<<32
	d3 = uint64(uint32(d3)+x14) | uint64(uint32(d3>>32)+x3)<<32
	d4 = uint64(uint32(d4)+x8) | uint64(uint32(d4>>32)+x13)<<32
	d5 = uint64(uint32(d5)+x2) | uint64(uint32(d5>>32)+x7)<<32
	d6 = uint64(uint32(d6)+x12) | uint64(uint32(d6>>32)+x1)<<32
	d7 = uint64(uint32(d7)+x6) | uint64(uint32(d7>>32)+x11)<<32

	out[0], tmp[0] = d0, d0
	out[1], tmp[1] = d1, d1
	out[2], tmp[2] = d2, d2
	out[3], tmp[3] = d3, d3
	out[4], tmp[4] = d4, d4
	out[5], tmp[5] = d5, d5
	out[6], tmp[6] = d6, d6
	out[7], tmp[7] = d7, d7
}

func blockMix(tmp *[8]uint64, in, out []uint64, r int) {
	blockCopy(tmp[:], in[(2*r-1)*8:], 8)
	for i := 0; i < 2*r; i += 2 {
		salsaXOR(tmp, in[i*8:], out[i*4:], 8)
		salsaXOR(tmp, in[i*8+8:], out[i*4+r*8:], 8)
	}
}

// These were tunable at design time, but they must meet certain constraints
const (
	PWXsimple = 2
	PWXgather = 4
	PWXrounds = 6
	Swidth    = 8
)

// Derived values.  These were never tunable on their own.
const (
	PWXbytes = PWXgather * PWXsimple * 8
	PWXwords = PWXbytes / 8
	Sbytes   = 3 * (1 << Swidth) * PWXsimple * 8
	Swords   = Sbytes / 8
	Smask    = (((1 << Swidth) - 1) * PWXsimple * 8)
)

type pwxformCtx struct {
	S0, S1, S2 []uint64
	w          uint32
}

func pwxform(X *[PWXwords]uint64, ctx *pwxformCtx) {
	S0, S1, S2, w := ctx.S0, ctx.S1, ctx.S2, ctx.w

	for i := 0; i < PWXrounds; i++ {
		for j := 0; j < PWXgather; j++ {
			// Unrolled inner loop for PWXsimple=2
			x := X[j*PWXsimple]
			xl := uint32(x)
			xh := uint32(x >> 32)
			x = uint64(xh) * uint64(xl)
			xl = (xl & Smask) / 8
			xh = (xh & Smask) / 8
			x = (x + S0[xl]) ^ S1[xh]
			X[j*PWXsimple] = x
			y := X[j*PWXsimple+1]
			y = ((y>>32)*uint64(uint32(y)) + S0[xl+1]) ^ S1[xh+1]
			X[j*PWXsimple+1] = y
			if i != 0 && i != PWXrounds-1 {
				S2[w] = x
				S2[w+1] = y
				w += 2
			}
		}
	}

	ctx.S0, ctx.S1, ctx.S2 = S2, S0, S1
	ctx.w = w & ((1<<Swidth)*PWXsimple - 1)
}

func blockMixPwxform(X *[PWXwords]uint64, B []uint64, r int, ctx *pwxformCtx) {
	r1 := 128 * r / PWXbytes
	blockCopy(X[:], B[(r1-1)*PWXwords:], PWXwords)
	for i := 0; i < r1; i++ {
		blockXOR(X[:], B[i*PWXwords:], PWXwords)
		pwxform(X, ctx)
		blockCopy(B[i*PWXwords:], X[:], PWXwords)
	}
	i := (r1 - 1) * PWXbytes / 64
	*X = [PWXwords]uint64{} // We don't need the XOR, so set X to zeroes
	salsaXOR(X, B[i*PWXwords:], B[i*PWXwords:], 2)
}

func integer(b []uint64, r int) uint32 {
	j := (2*r - 1) * 8
	return uint32(b[j])
}

func p2floor(x uint32) uint32 {
	for x&(x-1) != 0 {
		x &= x - 1
	}
	return x
}

func wrap(x, i uint32) uint32 {
	n := p2floor(i)
	return (x & (n - 1)) + (i - n)
}

func smix(b []byte, r, N, Nloop int, v, xy []uint64, ctx *pwxformCtx) {
	var tmp [8]uint64
	R := 16 * r
	x := xy
	y := xy[R:]

	j := 0
	for i := 0; i < R; i++ {
		lo := binary.LittleEndian.Uint32(b[(j & ^63)|((j*5)&63):])
		j += 4
		hi := binary.LittleEndian.Uint32(b[(j & ^63)|((j*5)&63):])
		j += 4
		x[i] = uint64(lo) | uint64(hi)<<32
	}
	if ctx != nil {
		for i := 0; i < N; i++ {
			blockCopy(v[i*R:], x, R)
			if i > 1 {
				j := int(wrap(integer(x, r), uint32(i)))
				blockXOR(x, v[j*R:], R)
			}
			blockMixPwxform(&tmp, x, r, ctx)
		}
		for i := 0; i < Nloop; i++ {
			j := int(integer(x, r) & uint32(N-1))
			blockXOR(x, v[j*R:], R)
			blockCopy(v[j*R:], x, R)
			blockMixPwxform(&tmp, x, r, ctx)
		}
	} else {
		for i := 0; i < N; i += 2 {
			blockCopy(v[i*R:], x, R)
			blockMix(&tmp, x, y, r)

			blockCopy(v[(i+1)*R:], y, R)
			blockMix(&tmp, y, x, r)
		}
		for i := 0; i < Nloop; i += 2 {
			j := int(integer(x, r) & uint32(N-1))
			blockXOR(x, v[j*R:], R)
			blockMix(&tmp, x, y, r)

			j = int(integer(y, r) & uint32(N-1))
			blockXOR(y, v[j*R:], R)
			blockMix(&tmp, y, x, r)
		}
	}
	j = 0
	for _, v := range x[:R] {
		binary.LittleEndian.PutUint32(b[(j & ^63)|((j*5)&63):], uint32(v))
		j += 4
		binary.LittleEndian.PutUint32(b[(j & ^63)|((j*5)&63):], uint32(v>>32))
		j += 4
	}
}

func smixYescrypt(b []byte, r, N int, v, xy []uint64, passwordSha256 []byte) {
	var ctx pwxformCtx
	var S [Swords]uint64
	smix(b, 1, Sbytes/128, 0, S[:], xy, nil)
	ctx.S2 = S[:]
	ctx.S1 = S[(1<<Swidth)*PWXsimple:]
	ctx.S0 = S[(1<<Swidth)*PWXsimple*2:]
	h := hmac.New(sha256.New, b[64*(2*r-1):])
	h.Write(passwordSha256)
	copy(passwordSha256, h.Sum(nil))
	smix(b, r, N, ((N+2)/3+1) & ^1, v, xy, &ctx)
}

func deriveKey(password, salt []byte, N, r, p, keyLen int, isYescrypt bool) ([]byte, error) {
	if N <= 1 || N&(N-1) != 0 {
		return nil, errors.New("(ye)scrypt: N must be > 1 and a power of 2")
	}
	if r <= 0 {
		return nil, errors.New("(ye)scrypt: r must be > 0")
	}
	if isYescrypt && p != 1 {
		return nil, errors.New("yescrypt: p must be 1")
	}
	if p <= 0 {
		return nil, errors.New("scrypt: p must be > 0")
	}
	if uint64(r)*uint64(p) >= 1<<30 || r > maxInt/128/p || r > maxInt/256 || N > maxInt/128/r {
		return nil, errors.New("(ye)scrypt: parameters are too large")
	}

	ppassword := &password
	pass := 1
	prehash := []byte("yescrypt-prehash")

	v := make([]uint64, 16*N*r)
	var xy []uint64
	var key []byte

	if isYescrypt {
		xy = make([]uint64, 16*max(r, 2))
		if N/p >= 0x100 && N/p*r >= 0x20000 {
			pass = 0
			N >>= 6
		}
	} else {
		xy = make([]uint64, 32*r)
	}

	for pass <= 1 {
		if isYescrypt {
			if pass == 1 {
				prehash = prehash[:8]
			}
			h := hmac.New(sha256.New, prehash)
			h.Write(*ppassword)
			passwordSha256 := h.Sum(nil)
			ppassword = &passwordSha256
		}

		b := pbkdf2.Key(*ppassword, salt, 1, p*128*r, sha256.New)

		if isYescrypt {
			copy(*ppassword, b[:32])
			smixYescrypt(b, r, N, v, xy, *ppassword)
		} else {
			for i := 0; i < p; i++ {
				smix(b[i*128*r:], r, N, N, v, xy, nil)
			}
		}

		key = pbkdf2.Key(*ppassword, b, 1, max(keyLen, 32), sha256.New)

		if isYescrypt {
			if pass == 0 {
				copy(*ppassword, key[:32])
				N <<= 6
			} else {
				h1 := hmac.New(sha256.New, key[:32])
				h1.Write([]byte("Client Key"))
				h2 := sha256.New()
				h2.Write(h1.Sum(nil))
				copy(key, h2.Sum(nil))
			}
		}

		pass++
	}

	return key[:keyLen], nil
}

// Classic scrypt
//
// ScryptKey implements classic scrypt (not yescrypt).  It is compatible with
// the x/crypto scrypt module's Key.
//
// It derives a key from the password, salt and cost parameters, returning a
// byte slice of length keyLen that can be used as cryptographic key.
//
// N is a CPU/memory cost parameter, which must be a power of two greater than 1.
// r and p must satisfy r * p < 2³⁰. If the parameters do not satisfy the
// limits, the function returns a nil byte slice and an error.
//
// For example, you can get a derived key for e.g. AES-256 (which needs a
// 32-byte key) by doing:
//
//	dk, err := yescrypt.ScryptKey([]byte("some password"), salt, 32768, 8, 1, 32)
//
// The recommended parameters for interactive logins as of 2017 are N=32768, r=8
// and p=1. The parameters N, r, and p should be increased as memory latency and
// CPU parallelism increases; consider setting N to the highest power of 2 you
// can derive within 100 milliseconds. Remember to get a good random salt.
func ScryptKey(password, salt []byte, N, r, p, keyLen int) ([]byte, error) {
	return deriveKey(password, salt, N, r, p, keyLen, false)
}

// Native yescrypt
//
// Key is similar to ScryptKey, but computes native yescrypt assuming
// reference yescrypt's current default flags (as of yescrypt 1.1.0), p=1
// (which it currently requires), t=0, and no ROM.  Example usage:
//
//	dk, err := yescrypt.Key([]byte("some password"), salt, 32768, 8, 1, 32)
//
// The set of parameters accepted by Key will likely change in future versions
// of this Go module to support more yescrypt functionality.
func Key(password, salt []byte, N, r, p, keyLen int) ([]byte, error) {
	return deriveKey(password, salt, N, r, p, keyLen, true)
}
