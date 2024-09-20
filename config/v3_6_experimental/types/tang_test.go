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
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func TestTangValidation(t *testing.T) {
	tests := []struct {
		in  Tang
		out error
		at  path.ContextPath
	}{
		// happy path with no advertisement and healthy url
		{
			in: Tang{
				URL:        "http://example.com",
				Thumbprint: util.StrToPtr("abc"),
			},
			out: nil,
		},
		// test 1: invalid url scheme
		{
			in: Tang{
				URL:        "httasfdsafadsf",
				Thumbprint: util.StrToPtr("abc"),
			},
			out: errors.ErrInvalidScheme,
			at:  path.New("foo", "url"),
		},
		// null url
		{
			in: Tang{
				Thumbprint: util.StrToPtr("abc"),
			},
			out: errors.ErrInvalidScheme,
			at:  path.New("foo", "url"),
		},
		// null thumbprint
		{
			in: Tang{
				URL:        "http://example.com",
				Thumbprint: nil,
			},
			out: errors.ErrTangThumbprintRequired,
			at:  path.New("foo", "thumbprint"),
		},
		// Advertisement is valid json
		{
			in: Tang{
				URL:           "http://example.com",
				Thumbprint:    util.StrToPtr("abc"),
				Advertisement: util.StrToPtr("{\"payload\": \"eyJrZXlzIjogW3siYWxnIjogIkVTNTEyIiwgImt0eSI6ICJFQyIsICJjcnYiOiAiUC01MjEiLCAieCI6ICJBRGFNajJmazNob21CWTF5WElSQ21uRk92cmUzOFZjdHMwTnNHeDZ6RWNxdEVXcjh5ekhUMkhfa2hjNGpSa19FQWFLdjNrd2RjZ05sOTBLcGhfMGYyQ190IiwgInkiOiAiQUZ2d1UyeGJ5T1RydWo0V1NtcVlqN2wtcUVTZmhWakdCNTI1Q2d6d0NoZUZRRTBvb1o3STYyamt3NkRKQ05yS3VPUDRsSEhicm8tYXhoUk9MSXNJVExvNCIsICJrZXlfb3BzIjogWyJ2ZXJpZnkiXX0sIHsiYWxnIjogIkVDTVIiLCAia3R5IjogIkVDIiwgImNydiI6ICJQLTUyMSIsICJ4IjogIkFOZDVYcTFvZklUbTdNWG16OUY0VVRSYmRNZFNIMl9XNXczTDVWZ0w3b3hwdmpyM0hkLXNLNUVqd3A1V2swMnJMb3NXVUJjYkZyZEhjZFJTTVJoZlVFTFIiLCAieSI6ICJBRVVaVlVZWkFBY2hVcmdoX3poaTV3SUUzeTEycGwzeWhqUk5LcGpSdW9tUFhKaDhRaFhXRmRWZEtMUlEwX1lwUjNOMjNSUk1pU1lvWlg0Qm42QnlrQVBMIiwgImtleV9vcHMiOiBbImRlcml2ZUtleSJdfV19\", \"protected\": \"eyJhbGciOiJFUzUxMiIsImN0eSI6Imp3ay1zZXQranNvbiJ9\", \"signature\": \"APHfSyVzLwELwG0pMJyIP74gWvhHUvDtv0SESBxA2uOdSXq76IdWHW2xvCZDdlNan8pnqUvEedPZjf_vdKBw9MTXAPMkRxVnu64HepKwlrzzm_zG2R4CHpoCOsGgjH9-acYxg-Vha63oMojv3_bV0VHg-NbzNLaxietgYplstvcNIwkv\"}"),
			},
			out: nil,
		},
		// Advertisement is empty string
		{
			in: Tang{
				URL:           "http://example.com",
				Thumbprint:    util.StrToPtr("abc"),
				Advertisement: util.StrToPtr(""),
			},
			out: nil,
		},
		// Advertisement is not valid json
		{
			in: Tang{
				URL:           "http://example.com",
				Thumbprint:    util.StrToPtr("abc"),
				Advertisement: util.StrToPtr("{{"),
			},
			out: errors.ErrInvalidTangAdvertisement,
			at:  path.New("foo", "advertisement"),
		},
	}
	for i, test := range tests {
		r := test.in.Validate(path.New("foo"))
		expected := report.Report{}
		expected.AddOnError(test.at, test.out)
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("#%d: bad error: expected : %v, got %v", i, expected, r)
		}
	}
}
