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

package types

import (
	"net/http"

	"github.com/coreos/ignition/config/shared/errors"
)

// Parse generates standard net/http headers from the data in HTTPHeaders
func (h HTTPHeaders) Parse() (http.Header, error) {
	headers := http.Header{}
	found := make(map[string]struct{})
	for _, header := range h {
		// Header name can't be empty
		if header.Name == "" {
			return nil, errors.ErrEmptyHTTPHeaderName
		}
		// Header names must be unique
		if _, ok := found[header.Name]; ok {
			return nil, errors.ErrDuplicateHTTPHeaders
		}
		found[header.Name] = struct{}{}
		headers.Add(header.Name, header.Value)
	}
	return headers, nil
}
