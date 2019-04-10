// Copyright 2019 Red Hat, Inc.
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

package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coreos/ignition/v2/config/translate/tests/pkga"
	"github.com/coreos/ignition/v2/config/translate/tests/pkgb"
)

// Note: we need different input and output types which unfortunately means a lot of tests

func TestTranslateTrivial(t *testing.T) {
	in := pkga.Trivial{
		A: "asdf",
		B: 5,
		C: true,
	}

	expected := pkgb.Trivial{
		A: "asdf",
		B: 5,
		C: true,
	}

	got := pkgb.Trivial{}

	trans := NewTranslator()

	trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
}

func TestTranslateNested(t *testing.T) {
	in := pkga.Nested{
		D: "foobar",
		Trivial: pkga.Trivial{
			A: "asdf",
			B: 5,
			C: true,
		},
	}

	expected := pkgb.Nested{
		D: "foobar",
		Trivial: pkgb.Trivial{
			A: "asdf",
			B: 5,
			C: true,
		},
	}

	got := pkgb.Nested{}

	trans := NewTranslator()

	trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
}

func TestTranslateTrivialReordered(t *testing.T) {
	in := pkga.TrivialReordered{
		A: "asdf",
		B: 5,
		C: true,
	}

	expected := pkgb.TrivialReordered{
		A: "asdf",
		B: 5,
		C: true,
	}

	got := pkgb.TrivialReordered{}

	trans := NewTranslator()

	trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
}

func TestCustomTranslatorTrivial(t *testing.T) {
	tr := func(a pkga.Trivial) pkgb.Nested {
		return pkgb.Nested{
			Trivial: pkgb.Trivial{
				A: a.A,
				B: a.B,
				C: a.C,
			},
			D: "abc",
		}
	}
	in := pkga.Trivial{
		A: "asdf",
		B: 5,
		C: true,
	}

	expected := pkgb.Nested{
		D: "abc",
		Trivial: pkgb.Trivial{
			A: "asdf",
			B: 5,
			C: true,
		},
	}

	got := pkgb.Nested{}

	trans := NewTranslator()
	trans.AddCustomTranslator(tr)

	trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
}

func TestCustomTranslatorTrivialWithAutomaticResume(t *testing.T) {
	trans := NewTranslator()
	tr := func(a pkga.Trivial) pkgb.Nested {
		ret := pkgb.Nested{
			D: "abc",
		}
		trans.Translate(&a, &ret.Trivial)
		return ret
	}
	in := pkga.Trivial{
		A: "asdf",
		B: 5,
		C: true,
	}

	expected := pkgb.Nested{
		D: "abc",
		Trivial: pkgb.Trivial{
			A: "asdf",
			B: 5,
			C: true,
		},
	}

	got := pkgb.Nested{}

	trans.AddCustomTranslator(tr)

	trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
}

func TestCustomTranslatorList(t *testing.T) {
	tr := func(a pkga.Trivial) pkgb.Nested {
		return pkgb.Nested{
			Trivial: pkgb.Trivial{
				A: a.A,
				B: a.B,
				C: a.C,
			},
			D: "abc",
		}
	}
	in := pkga.HasList{
		L: []pkga.Trivial{
			{
				A: "asdf",
				B: 5,
				C: true,
			},
		},
	}

	expected := pkgb.HasList{
		L: []pkgb.Nested{
			{
				D: "abc",
				Trivial: pkgb.Trivial{
					A: "asdf",
					B: 5,
					C: true,
				},
			},
		},
	}

	got := pkgb.HasList{}

	trans := NewTranslator()
	trans.AddCustomTranslator(tr)

	trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
}
