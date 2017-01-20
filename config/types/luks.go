// Copyright 2017 CoreOS, Inc.
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

	"github.com/coreos/ignition/config/validate/report"
)

var (
	ErrLuksNoDevice = errors.New("no device provided for encrypted volume setup")
	ErrLuksNoKey    = errors.New("no key provided for encrypted volume setup")
)

type Luks struct {
	Name   string `json:"name,omitempty"`
	Device Path   `json:"device,omitempty"`
	Key    string `json:"key,omitempty"`
}

func (l Luks) Validate() report.Report {
	if len(l.Device) == 0 {
		return report.ReportFromError(ErrLuksNoDevice, report.EntryError)
	}
	if l.Key == "" {
		return report.ReportFromError(ErrLuksNoKey, report.EntryError)
	}

	return report.Report{}
}
