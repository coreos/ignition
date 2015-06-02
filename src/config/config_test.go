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

package config

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	type in struct {
		config []byte
	}
	type out struct {
		config Config
		err    error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{config: []byte(`{"version": 1}`)},
			out: out{config: Config{Version: 1}},
		},
		{
			in:  in{config: []byte(`{}`)},
			out: out{err: ErrVersion},
		},
		{
			in:  in{config: []byte(`#cloud-config`)},
			out: out{err: ErrCloudConfig},
		},
		{
			in:  in{config: []byte(`#!/bin/sh`)},
			out: out{err: ErrScript},
		},
	}

	for i, test := range tests {
		config, err := Parse(test.in.config)
		if !reflect.DeepEqual(test.out.config, config) {
			t.Errorf("#%d: bad config: want %+v, got %+v", i, test.out.config, config)
		}
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.err, err)
		}
	}
}
