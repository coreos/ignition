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
	"strings"

	"github.com/coreos/ignition/config/shared/errors"
	"github.com/coreos/ignition/config/validate/report"
)

func (s Storage) MergedKeys() map[string]string {
	return map[string]string{
		"Directories": "Node",
		"Files":       "Node",
		"Links":       "Node",
	}
}

func (s Storage) Validate() (r report.Report) {
	for _, d := range s.Directories {
		for _, l := range s.Links {
			if strings.HasPrefix(d.Path, l.Path+"/") {
				r.AddOnError(errors.ErrDirectoryUsedSymlink)
			}
		}
	}
	for _, f := range s.Files {
		for _, l := range s.Links {
			if strings.HasPrefix(f.Path, l.Path+"/") {
				r.AddOnError(errors.ErrFileUsedSymlink)
			}
		}
	}
	for _, l1 := range s.Links {
		for _, l2 := range s.Links {
			if strings.HasPrefix(l1.Path, l2.Path+"/") {
				r.AddOnError(errors.ErrLinkUsedSymlink)
			}
		}
	}
	return
}
