// Copyright 2024 Red Hat
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

package partitioners

import (
	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
)

type DeviceManager interface {
	CreatePartition(p Partition)
	DeletePartition(num int)
	Info(num int)
	WipeTable(wipe bool)
	Pretend() (string, error)
	Commit() error
	ParseOutput(string, []int) (map[int]Output, error)
}

type Partition struct {
	types.Partition
	StartSector   *int64
	SizeInSectors *int64
	StartMiB      string
	SizeMiB       string
}

type Output struct {
	Start int64
	Size  int64
}
