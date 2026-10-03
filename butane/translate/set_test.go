// Copyright 2022 Red Hat, Inc.
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

	"github.com/coreos/vcontext/path"
	"github.com/stretchr/testify/assert"
)

// mkTrans makes a TranslationSet with no tag in the paths consuming pairs of args. i.e:
// mkTrans(from1, to1, from2, to2) -> a set wiht from1->to1, from2->to2
// This is just a shorthand for making writing tests easier
func mkTrans(paths ...path.ContextPath) TranslationSet {
	ret := TranslationSet{Set: map[string]Translation{}}
	if len(paths)%2 == 1 {
		panic("Odd number of args to mkTrans")
	}
	for i := 0; i < len(paths); i += 2 {
		ret.AddTranslation(paths[i], paths[i+1])
	}
	return ret
}

// fp means "fastpath"; super shorthand, we'll use it a lot
func fp(parts ...interface{}) path.ContextPath {
	return path.New("", parts...)
}

func TestTranslationSetMap(t *testing.T) {
	create := func() TranslationSet {
		return mkTrans(
			fp(), fp(),
			fp("a"), fp("A"),
			fp("a", 0), fp("A", 0),
			fp("a", 0, "b"), fp("A", 0, "B"),
			fp("a", 0, "b", "c"), fp("A", 0, "B"),
			fp("a", 0, "b", "d"), fp("A", 0, "B", 0),
			fp("a", 0, "b", "e"), fp("A", 0, "B", 0, "C"),
			fp("a", 0, "b", "f"), fp("A", 0, "B", 0, "D"),
			fp("clobbered"), fp("A", 0, "B", 0, "G"),
			fp("a", 0, "b", "g"), fp("A", 0, "B", 1),
			fp("a", 0, "b", "h"), fp("A", 0, "B", 1, "E"),
			fp("a", 0, "b", "i"), fp("A", 0, "B", 1, "F"),
		)
	}
	ts := create()
	result := ts.Map(mkTrans(
		fp("A", 0, "B", 0, "C"), fp("A", 0, "B", 0, "G"),
		fp("A", 0, "B", 0, "D"), fp("A", 0, "H"),
		fp("missing"), fp("B"),
	))
	assert.Equal(t, create(), ts, "original was changed")
	assert.Equal(t, mkTrans(
		fp(), fp(),
		fp("a"), fp("A"),
		fp("a", 0), fp("A", 0),
		fp("a", 0, "b"), fp("A", 0, "B"),
		fp("a", 0, "b", "c"), fp("A", 0, "B"),
		fp("a", 0, "b", "d"), fp("A", 0, "B", 0),
		fp("a", 0, "b", "e"), fp("A", 0, "B", 0, "G"),
		fp("a", 0, "b", "f"), fp("A", 0, "H"),
		fp("a", 0, "b", "g"), fp("A", 0, "B", 1),
		fp("a", 0, "b", "h"), fp("A", 0, "B", 1, "E"),
		fp("a", 0, "b", "i"), fp("A", 0, "B", 1, "F"),
	), result, "bad mapping")
}

func TestTranslationSetAddFromCommonSource(t *testing.T) {
	type Sub struct {
		C int `json:"c"`
	}
	type Main struct {
		A string `json:"a"`
		B Sub    `json:"b"`
	}

	expected := NewTranslationSet("yaml", "json")
	expected.AddTranslation(path.New("yaml", "y"), path.New("json", "z", 0))
	expected.AddTranslation(path.New("yaml", "y", "a"), path.New("json", "z", 0, "a"))
	expected.AddTranslation(path.New("yaml", "y", "b"), path.New("json", "z", 0, "b"))
	expected.AddTranslation(path.New("yaml", "y", "b", "c"), path.New("json", "z", 0, "b", "c"))

	actual := NewTranslationSet("yaml", "json")
	actual.AddFromCommonObject(path.New("yaml", "y"), path.New("json", "z", 0), &Main{})
	assert.Equal(t, expected, actual)
}
