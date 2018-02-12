// Copyright 2016 CoreOS, Inc.
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
	"errors"
	"net/url"

	"github.com/vincent-petithory/dataurl"
)

var (
	// ErrInvalidScheme is returned when the scheme is not valid for the context.
	ErrInvalidScheme = errors.New("invalid url scheme")
)

func validateURL(s string, subset ...string) error {
	// Empty url is valid, indicates an empty file
	if s == "" {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "http", "https", "oem", "tftp", "s3":
		break
	case "data":
		if _, err := dataurl.DecodeString(s); err != nil {
			return err
		}
		break
	default:
		return ErrInvalidScheme
	}

	if len(subset) != 0 {
		for _, allowed := range subset {
			if u.Scheme == allowed {
				return nil
			}
		}
		return ErrInvalidScheme
	}

	return nil
}
