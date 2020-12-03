// Copyright 2020 Red Hat, Inc.
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

package v3_3_experimental

import (
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	type in struct {
		config []byte
	}
	type out struct {
		config types.Config
		err    error
	}

	tests := []struct {
		in  in
		out out
	}{
		{
			in:  in{config: []byte(`{"ignitionVersion": 1}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "1.0.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.0.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.1.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.2.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.3.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.4.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.0.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.1.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.2.0"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.0.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.1.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.2.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.3.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.4.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.5.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.0.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.1.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.2.0-experimental"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "3.3.0-experimental"}}`)},
			out: out{config: types.Config{Ignition: types.Ignition{Version: types.MaxVersion.String()}}},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "2.0.0"},}`)},
			out: out{err: errors.ErrInvalid},
		},
		{
			in:  in{config: []byte(`{"ignition": {"version": "invalid.semver"}}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte(`{}`)},
			out: out{err: errors.ErrUnknownVersion},
		},
		{
			in:  in{config: []byte{}},
			out: out{err: errors.ErrEmpty},
		},
	}

	for i, test := range tests {
		config, report, err := Parse(test.in.config)
		if test.out.err != err {
			t.Errorf("#%d: bad error: want %v, got %v, report: %+v", i, test.out.err, err, report)
		}
		assert.Equal(t, test.out.config, config, "#%d: bad config, report: %+v", i, report)
	}
}
