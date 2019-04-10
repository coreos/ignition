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

package files

import (
	"reflect"
	"sort"
	"testing"

	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
)

type pathWrapper string

func (pw pathWrapper) getPath() string {
	return string(pw)
}

func (pw pathWrapper) create(l *log.Logger, u util.Util) error {
	return nil
}

func TestEntrySort(t *testing.T) {
	type in struct {
		data []string
	}

	type out struct {
		data []string
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{data: []string{"/a/b/c/d/e/", "/a/b/c/d/", "/a/b/c/", "/a/b/", "/a/"}},
			out: out{data: []string{"/a/", "/a/b/", "/a/b/c/", "/a/b/c/d/", "/a/b/c/d/e/"}},
		},
		{
			in:  in{data: []string{"/a////b/c/d/e/", "/", "/a/b/c//d/", "/a/b/c/", "/a/b/", "/a/"}},
			out: out{data: []string{"/", "/a/", "/a/b/", "/a/b/c/", "/a/b/c//d/", "/a////b/c/d/e/"}},
		},
		{
			in:  in{data: []string{"/a/", "/a/../a/b", "/"}},
			out: out{data: []string{"/", "/a/", "/a/../a/b"}},
		},
	}

	for i, test := range tests {
		dirs := make([]pathWrapper, len(test.in.data))
		for j := range dirs {
			dirs[j] = pathWrapper(test.in.data[j])
		}
		sort.Slice(dirs, func(i, j int) bool { return util.Depth(dirs[i].getPath()) < util.Depth(dirs[j].getPath()) })
		outpaths := make([]string, len(test.in.data))
		for j, dir := range dirs {
			outpaths[j] = dir.getPath()
		}
		if !reflect.DeepEqual(test.out.data, outpaths) {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.data, outpaths)
		}
	}
}
