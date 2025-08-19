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
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

var (
	MaxVersion = semver.Version{
		Major:      3,
		Minor:      6,
		PreRelease: "experimental",
	}
)

func (cfg Config) Validate(c path.ContextPath) (r report.Report) {
	systemdPath := "/etc/systemd/system/"
	unitPaths := map[string]struct{}{}
	for _, unit := range cfg.Systemd.Units {
		if !util.NilOrEmpty(unit.Contents) {
			pathString := systemdPath + unit.Name
			unitPaths[pathString] = struct{}{}
		}
		for _, dropin := range unit.Dropins {
			if !util.NilOrEmpty(dropin.Contents) {
				pathString := systemdPath + unit.Name + ".d/" + dropin.Name
				unitPaths[pathString] = struct{}{}
			}
		}
	}
	for i, f := range cfg.Storage.Files {
		if _, exists := unitPaths[f.Path]; exists {
			r.AddOnError(c.Append("storage", "files", i, "path"), errors.ErrPathConflictsSystemd)
		}
	}
	for i, d := range cfg.Storage.Directories {
		if _, exists := unitPaths[d.Path]; exists {
			r.AddOnError(c.Append("storage", "directories", i, "path"), errors.ErrPathConflictsSystemd)
		}
	}
	for i, l := range cfg.Storage.Links {
		if _, exists := unitPaths[l.Path]; exists {
			r.AddOnError(c.Append("storage", "links", i, "path"), errors.ErrPathConflictsSystemd)
		}
	}

	r.Merge(cfg.validateParents(c))

	return r
}

func (cfg Config) validateParents(c path.ContextPath) report.Report {
	type entry struct {
		Path  string
		Field string
		Index int
	}

	var entries []entry
	paths := make(map[string]entry)
	r := report.Report{}

	for i, f := range cfg.Storage.Files {
		if _, exists := paths[f.Path]; exists {
			r.AddOnError(c.Append("storage", "files", i, "path"), errors.ErrPathAlreadyExists)
		} else {
			paths[f.Path] = entry{Path: f.Path, Field: "files", Index: i}
			entries = append(entries, entry{Path: f.Path, Field: "files", Index: i})
		}
	}

	for i, d := range cfg.Storage.Directories {
		if _, exists := paths[d.Path]; exists {
			r.AddOnError(c.Append("storage", "directories", i, "path"), errors.ErrPathAlreadyExists)
		} else {
			paths[d.Path] = entry{Path: d.Path, Field: "directories", Index: i}
			entries = append(entries, entry{Path: d.Path, Field: "directories", Index: i})
		}
	}

	for i, l := range cfg.Storage.Links {
		if _, exists := paths[l.Path]; exists {
			r.AddOnError(c.Append("storage", "links", i, "path"), errors.ErrPathAlreadyExists)
		} else {
			paths[l.Path] = entry{Path: l.Path, Field: "links", Index: i}
			entries = append(entries, entry{Path: l.Path, Field: "links", Index: i})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return len(strings.Split(strings.Trim(entries[i].Path, "/"), "/")) < len(strings.Split(strings.Trim(entries[j].Path, "/"), "/"))
	})

	// Check for path conflicts where a file/link is used as a parent directory
	for _, entry := range entries {
		// Skip root path
		if entry.Path == "/" {
			continue
		}

		// Check if any parent path exists as a file or link
		parentPath := entry.Path
		for {
			parentPath = filepath.Dir(parentPath)
			if parentPath == "/" || parentPath == "." {
				break
			}

			if parentEntry, exists := paths[parentPath]; exists {
				// If the parent is not a directory, it's a conflict
				if parentEntry.Field != "directories" {
					errorMsg := fmt.Errorf("invalid entry at path %s: %w", entry.Path, errors.ErrMissLabeledDir)
					r.AddOnError(c.Append("storage", entry.Field, entry.Index, "path"), errorMsg)
					break
				}
			}
		}
	}

	return r
}
