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
	"path"

	"github.com/coreos/ignition/v2/config/shared/errors"

	cpath "github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func (u Unit) Key() string {
	return u.Name
}

func (d Dropin) Key() string {
	return d.Name
}

func (u Unit) Validate(c cpath.ContextPath) (r report.Report) {
	var err error
	r.AddOnError(c.Append("name"), validateName(u.Name))
	r.Merge(u.Contents.Validate(c))
	c = c.Append("contents")
	r.AddOnError(c, err)

	return
}

func validateName(name string) error {
	switch path.Ext(name) {
	case ".service", ".socket", ".device", ".mount", ".automount", ".swap", ".target", ".path", ".timer", ".snapshot", ".slice", ".scope":
	default:
		return errors.ErrInvalidSystemdExt
	}
	return nil
}

func (d Dropin) Validate(c cpath.ContextPath) (r report.Report) {
	switch path.Ext(d.Name) {
	case ".conf":
	default:
		r.AddOnError(c.Append("name"), errors.ErrInvalidSystemdDropinExt)
	}

	return
}
