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

package validate

import (
	"fmt"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/validate/report"
)

// rule is a function which performs whole-config checks the need to be performed
// after parsing the whole config
type rule func(cfg types.Config, report *report.Report)

func applyRules(cfg types.Config) (report report.Report) {
	rules := []rule{
		checkFilesFilesystems,
		checkDuplicateFilesystems,
	}

	for _, r := range rules {
		r(cfg, &report)
	}
	return
}

func checkFilesFilesystems(cfg types.Config, r *report.Report) {
	filesystems := map[string]struct{}{"root": {}}
	for _, filesystem := range cfg.Storage.Filesystems {
		filesystems[filesystem.Name] = struct{}{}
	}
	for _, file := range cfg.Storage.Files {
		if file.Filesystem == "" {
			// Filesystem was not specified. This is an error, but its handled in types.File's AssertValid, not here
			continue
		}
		_, ok := filesystems[file.Filesystem]
		if !ok {
			r.Add(report.Entry{
				Kind: report.EntryWarning,
				Message: fmt.Sprintf("File %q references nonexistent filesystem %q. (This is ok if it is defined in a referenced config)",
					file.Path, file.Filesystem),
			})
		}
	}
}

func checkDuplicateFilesystems(cfg types.Config, r *report.Report) {
	filesystems := map[string]struct{}{"root": {}}
	for _, filesystem := range cfg.Storage.Filesystems {
		if _, ok := filesystems[filesystem.Name]; ok {
			r.Add(report.Entry{
				Kind:    report.EntryWarning,
				Message: fmt.Sprintf("Filesystem %q shadows exising filesystem definition", filesystem.Name),
			})
		}
		filesystems[filesystem.Name] = struct{}{}
	}
}
