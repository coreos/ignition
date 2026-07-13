// Copyright 2020 Red Hat, Inc
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
// limitations under the License.)

package v0_8_exp

import (
	"strings"

	baseutil "github.com/coreos/butane/base/util"
	"github.com/coreos/butane/config/common"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func (rs Resource) Validate(c path.ContextPath) (r report.Report) {
	var field string
	sources := 0
	// Local files are validated in the translateResource function
	if rs.Local != nil {
		sources++
		field = "local"
	}
	if rs.Inline != nil {
		sources++
		field = "inline"
	}
	if rs.Source != nil {
		sources++
		field = "source"
	}
	if sources > 1 {
		r.AddOnError(c.Append(field), common.ErrTooManyResourceSources)
		return
	}
	if strings.HasPrefix(c.String(), "$.ignition.config") {
		if field == "inline" {
			rp, err := ValidateIgnitionConfig(c, []byte(*rs.Inline))
			r.Merge(rp)
			if err != nil {
				r.AddOnError(c.Append(field), err)
				return
			}
		}
	}
	return
}

func (fs Filesystem) Validate(c path.ContextPath) (r report.Report) {
	if !util.IsTrue(fs.WithMountUnit) {
		return
	}
	if util.NilOrEmpty(fs.Format) {
		r.AddOnError(c.Append("format"), common.ErrMountUnitNoFormat)
	} else if *fs.Format != "swap" && util.NilOrEmpty(fs.Path) {
		r.AddOnError(c.Append("path"), common.ErrMountUnitNoPath)
	}
	return
}

func (d Directory) Validate(c path.ContextPath) (r report.Report) {
	if d.Mode != nil {
		r.AddOnWarn(c.Append("mode"), baseutil.CheckForDecimalMode(*d.Mode, true))
	}
	return
}

func (f File) Validate(c path.ContextPath) (r report.Report) {
	if f.Mode != nil {
		r.AddOnWarn(c.Append("mode"), baseutil.CheckForDecimalMode(*f.Mode, false))
	}
	return
}

func (t Tree) Validate(c path.ContextPath) (r report.Report) {
	if t.Local == "" {
		r.AddOnError(c, common.ErrTreeNoLocal)
	}
	return
}

func validateNotTooManySources(contentsLocal, contents *string, c path.ContextPath) (r report.Report) {
	if contentsLocal != nil && contents != nil {
		r.AddOnError(c.Append("contents_local"), common.ErrTooManySystemdSources)
	}
	return
}

func (rs Unit) Validate(c path.ContextPath) (r report.Report) {
	return validateNotTooManySources(rs.ContentsLocal, rs.Contents, c)
}

func (rs Dropin) Validate(c path.ContextPath) (r report.Report) {
	return validateNotTooManySources(rs.ContentsLocal, rs.Contents, c)
}

// All accepted extensions by podman-systemd.unit
func validateQuadletExtension(name string) error {
	extensionIsSupported := strings.HasSuffix(name, ".container") ||
		strings.HasSuffix(name, ".volume") ||
		strings.HasSuffix(name, ".network") ||
		strings.HasSuffix(name, ".kube") ||
		strings.HasSuffix(name, ".image") ||
		strings.HasSuffix(name, ".build") ||
		strings.HasSuffix(name, ".pod") ||
		strings.HasSuffix(name, ".artifact")

	if !extensionIsSupported {
		return common.ErrQuadletBadExtension
	}

	return nil
}

// Validate checks the quadlet name has a valid extension and template instances don't have contents.
func (rs Quadlet) Validate(c path.ContextPath) (r report.Report) {
	if err := validateQuadletExtension(rs.Name); err != nil {
		r.AddOnError(c.Append("name"), err)
		return r
	}

	// Template instances cannot have a content as they are symlinks, and non-template instances
	// can have either a contents or a contents_local, but not both
	if isTemplate, _ := isTemplateInstance(rs.Name); isTemplate {
		if rs.Contents != nil {
			contentPath := c.Append("contents")
			r.AddOnError(contentPath, common.ErrTemplateInstanceCannotHaveContents)
		}
		if rs.ContentsLocal != nil {
			contentPath := c.Append("contents_local")
			r.AddOnError(contentPath, common.ErrTemplateInstanceCannotHaveContents)
		}
	} else {
		r.Merge(validateNotTooManySources(rs.ContentsLocal, rs.Contents, c))
	}
	return r
}
