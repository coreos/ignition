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
	"fmt"

	"github.com/coreos/ignition/config/validate/report"
)

var (
	ErrAppendAndOverwrite = errors.New("cannot set both append and overwrite to true")
	ErrCompressionInvalid = errors.New("invalid compression method")
)

func (f File) Validate() report.Report {
	if f.Overwrite != nil && *f.Overwrite && f.Append {
		return report.ReportFromError(ErrAppendAndOverwrite, report.EntryError)
	}
	return report.Report{}
}

func (f File) ValidateMode() report.Report {
	r := report.Report{}
	if err := validateMode(f.Mode); err != nil {
		r.Add(report.Entry{
			Message: err.Error(),
			Kind:    report.EntryError,
		})
	}
	if f.Mode == nil {
		r.Add(report.Entry{
			Message: "file permissions unset, defaulting to 0000",
			Kind:    report.EntryWarning,
		})
	}
	return r
}

func (fc FileContents) ValidateCompression() report.Report {
	r := report.Report{}
	switch fc.Compression {
	case "", "gzip":
	default:
		r.Add(report.Entry{
			Message: ErrCompressionInvalid.Error(),
			Kind:    report.EntryError,
		})
	}
	return r
}

func (fc FileContents) ValidateSource() report.Report {
	r := report.Report{}
	err := validateURL(fc.Source)
	if err != nil {
		r.Add(report.Entry{
			Message: fmt.Sprintf("invalid url %q: %v", fc.Source, err),
			Kind:    report.EntryError,
		})
	}
	return r
}
