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

package types

import (
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
)

func TestURLValidate(t *testing.T) {
	tests := []struct {
		in  *string
		out error
	}{
		{
			nil,
			nil,
		},
		{
			util.StrToPtr(""),
			nil,
		},
		{
			util.StrToPtr("http://example.com"),
			nil,
		},
		{
			util.StrToPtr("https://example.com"),
			nil,
		},
		{
			util.StrToPtr("tftp://example.com:69/foobar.txt"),
			nil,
		},
		{
			util.StrToPtr("data:,example%20file%0A"),
			nil,
		},
		{
			util.StrToPtr("bad://"),
			errors.ErrInvalidScheme,
		},
		{
			util.StrToPtr("s3://bucket/key"),
			nil,
		},
		{
			util.StrToPtr("s3://bucket/key?versionId="),
			errors.ErrInvalidS3ObjectVersionId,
		},
		{
			util.StrToPtr("s3://bucket/key?versionId=aVersionHash"),
			nil,
		},
		{
			util.StrToPtr("gs://bucket/object"),
			nil,
		},
	}

	for i, test := range tests {
		err := validateURLNilOK(test.in)
		if test.out != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, err)
		}
	}
}
