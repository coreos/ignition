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
//
// +build s390x

package dasdfmt

import (
	"fmt"
	"os/exec"

	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
)

// Formats DASD disk. dasdfmt tool is used to format DASDs.
// https://www.ibm.com/support/knowledgecenter/linuxonibm/com.ibm.linux.z.lgdd/lgdd_r_dasdfmt_cmd.html
func Format(logger *log.Logger, dev string) error {
	// --blocksize : 4096 for DASD has 4k sectors
	// --disk_layout: cdl is the only supported layout
	// --mode: full formats entire disk
	// -y  : starts without confirmation
	// -v  : verbose mode
	opts := []string{"--blocksize=4096", "--disk_layout=cdl", "--mode=full", "-yv", dev}
	logger.Info("running dasdfmt with options: %v", opts)
	cmd := exec.Command(distro.DasdfmtCmd(), opts...)
	if _, err := logger.LogCmd(cmd, "formatting disk %q", dev); err != nil {
		return fmt.Errorf("formatting failed: %v", err)
	}

	return nil
}
