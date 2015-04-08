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

package providers

import (
	"fmt"
	"strings"
)

type List []string

func (s List) String() string {
	return strings.Join(s, ",")
}

func (s *List) Set(val string) error {
	if provider := Get(val); provider == nil {
		return fmt.Errorf("%s is not a valid config provider", val)
	}

	*s = append(*s, val)
	return nil
}
