// Copyright 2021 Red Hat, Inc
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

package v4_21

import (
	"slices"

	"github.com/coreos/butane/config/common"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func (m Metadata) Validate(c path.ContextPath) (r report.Report) {
	if m.Name == "" {
		r.AddOnError(c.Append("name"), common.ErrNameRequired)
	}
	if m.Labels[ROLE_LABEL_KEY] == "" {
		r.AddOnError(c.Append("labels"), common.ErrRoleRequired)
	}
	return
}

func (os OpenShift) Validate(c path.ContextPath) (r report.Report) {
	if os.KernelType != nil {
		switch *os.KernelType {
		case "", "default", "realtime":
		default:
			r.AddOnError(c.Append("kernel_type"), common.ErrInvalidKernelType)
		}
	}
	return
}

// Validate that we have the required kernel argument pointing to the key file
// if we have CEX support enabled. We only do this in the openshift spec as
// this is implemented differently in the fcos one.
// See: https://github.com/coreos/butane/issues/611
// See: https://github.com/coreos/butane/issues/613
func (conf Config) Validate(c path.ContextPath) (r report.Report) {
	if util.IsTrue(conf.BootDevice.Luks.Cex.Enabled) && !slices.Contains(conf.OpenShift.KernelArguments, "rd.luks.key=/etc/luks/cex.key") {
		r.AddOnError(c.Append("openshift", "kernel_arguments"), common.ErrMissingKernelArgumentCex)
	}
	cex := false
	for _, l := range conf.Storage.Luks {
		if util.IsTrue(l.Cex.Enabled) && l.Name == "root" {
			cex = true
		}
	}
	if cex && !slices.Contains(conf.OpenShift.KernelArguments, "rd.luks.key=/etc/luks/cex.key") {
		r.AddOnError(c.Append("openshift", "kernel_arguments"), common.ErrMissingKernelArgumentCex)
	}

	return
}
