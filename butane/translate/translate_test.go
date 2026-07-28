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
	"errors"
	"testing"

	"github.com/coreos/butane/translate/tests/pkga"
	"github.com/coreos/butane/translate/tests/pkgb"

	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

type testOptions struct{}

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
	exTrans := mkTrans(
		fp(), fp(),
		fp("A"), fp("A"),
		fp("B"), fp("B"),
		fp("C"), fp("C"),
	)

	got := pkgb.Trivial{}

	trans := NewTranslator("", "", testOptions{})

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "", "non-empty report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
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
	exTrans := mkTrans(
		fp(), fp(),
		fp("A"), fp("A"),
		fp("B"), fp("B"),
		fp("C"), fp("C"),
		fp("D"), fp("D"),
	)

	got := pkgb.Nested{}

	trans := NewTranslator("", "", testOptions{})

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "", "non-empty report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
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
	exTrans := mkTrans(
		fp(), fp(),
		fp("A"), fp("A"),
		fp("B"), fp("B"),
		fp("C"), fp("C"),
	)

	got := pkgb.TrivialReordered{}

	trans := NewTranslator("", "", testOptions{})

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "", "non-empty report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
}

func TestTranslateTrivialSkip(t *testing.T) {
	in := pkga.TrivialSkip{
		A: "asdf",
		B: 5,
		C: true,
	}

	expected := pkgb.TrivialSkip{
		B: 5,
		C: true,
	}
	exTrans := mkTrans(
		fp(), fp(),
		fp("B"), fp("B"),
		fp("C"), fp("C"),
	)

	got := pkgb.TrivialSkip{}

	trans := NewTranslator("", "", testOptions{})

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "", "non-empty report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
}

func TestCustomTranslatorTrivial(t *testing.T) {
	tr := func(a pkga.Trivial, options testOptions) (pkgb.Nested, TranslationSet, report.Report) {
		ts := mkTrans(fp("A"), fp("A"),
			fp("B"), fp("B"),
			fp("C"), fp("C"),
			fp("C"), fp("D"),
		)
		var r report.Report
		r.AddOnInfo(fp("A"), errors.New("info"))
		return pkgb.Nested{
			Trivial: pkgb.Trivial{
				A: a.A,
				B: a.B,
				C: a.C,
			},
			D: "abc",
		}, ts, r
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
	exTrans := mkTrans(
		fp(), fp(),
		fp("A"), fp("A"),
		fp("B"), fp("B"),
		fp("C"), fp("C"),
		fp("C"), fp("D"),
	)

	got := pkgb.Nested{}

	trans := NewTranslator("", "", testOptions{})
	trans.AddCustomTranslator(tr)

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "info at $.A: info\n", "bad report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
}

func TestCustomTranslatorTrivialWithAutomaticResume(t *testing.T) {
	trans := NewTranslator("", "", testOptions{})
	tr := func(a pkga.Trivial, options testOptions) (pkgb.Nested, TranslationSet, report.Report) {
		ret := pkgb.Nested{
			D: "abc",
		}
		ts, r := trans.Translate(&a, &ret.Trivial)
		ts.AddTranslation(fp("C"), fp("D"))
		return ret, ts, r
	}
	in := pkga.Trivial{
		A: "asdf",
		B: 5,
		C: true,
	}
	exTrans := mkTrans(
		fp(), fp(),
		fp("A"), fp("A"),
		fp("B"), fp("B"),
		fp("C"), fp("C"),
		fp("C"), fp("D"),
	)

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

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "", "non-empty report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
}

func TestCustomTranslatorList(t *testing.T) {
	tr := func(a pkga.Trivial, options testOptions) (pkgb.Nested, TranslationSet, report.Report) {
		ts := mkTrans(fp("A"), fp("A"),
			fp("B"), fp("B"),
			fp("C"), fp("C"),
			fp("C"), fp("D"),
		)
		return pkgb.Nested{
			Trivial: pkgb.Trivial{
				A: a.A,
				B: a.B,
				C: a.C,
			},
			D: "abc",
		}, ts, report.Report{}
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
	exTrans := mkTrans(
		fp(), fp(),
		fp("L"), fp("L"),
		fp("L", 0), fp("L", 0),
		fp("L", 0, "A"), fp("L", 0, "A"),
		fp("L", 0, "B"), fp("L", 0, "B"),
		fp("L", 0, "C"), fp("L", 0, "C"),
		fp("L", 0, "C"), fp("L", 0, "D"),
	)

	got := pkgb.HasList{}

	trans := NewTranslator("", "", testOptions{})
	trans.AddCustomTranslator(tr)

	ts, r := trans.Translate(&in, &got)
	assert.Equal(t, got, expected, "bad translation")
	assert.Equal(t, ts, exTrans, "bad translation")
	assert.Equal(t, r.String(), "", "non-empty report")
	assert.NoError(t, ts.DebugVerifyCoverage(&got), "incomplete TranslationSet coverage")
}
