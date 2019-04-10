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
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/validate/report"
)

func (d Disk) Key() string {
	return d.Device
}

func (n Disk) Validate() report.Report {
	return report.Report{}
}

func (n Disk) ValidateDevice() (r report.Report) {
	if len(n.Device) == 0 {
		r.AddOnError(errors.ErrDiskDeviceRequired)
		return
	}
	r.AddOnError(validatePath(n.Device))
	return
}

func (n Disk) ValidatePartitions() (r report.Report) {
	if n.partitionNumbersCollide() {
		r.AddOnError(errors.ErrPartitionNumbersCollide)
	}
	if n.partitionsOverlap() {
		r.AddOnError(errors.ErrPartitionsOverlap)
	}
	if n.partitionsMixZeroesAndNonexistence() {
		r.AddOnError(errors.ErrZeroesWithShouldNotExist)
	}
	if n.partitionLabelsCollide() {
		r.AddOnError(errors.ErrDuplicateLabels)
	}
	return
}

// partitionNumbersCollide returns true if partition numbers in n.Partitions are not unique.
func (n Disk) partitionNumbersCollide() bool {
	m := map[int][]Partition{}
	for _, p := range n.Partitions {
		if p.Number != 0 {
			// a number of 0 means next available number, multiple devices can specify this
			m[p.Number] = append(m[p.Number], p)
		}
	}
	for _, n := range m {
		if len(n) > 1 {
			// TODO(vc): return information describing the collision for logging
			return true
		}
	}
	return false
}

func (d Disk) partitionLabelsCollide() bool {
	m := map[string]struct{}{}
	for _, p := range d.Partitions {
		if p.Label != nil {
			// a number of 0 means next available number, multiple devices can specify this
			if _, exists := m[*p.Label]; exists {
				return true
			}
			m[*p.Label] = struct{}{}
		}
	}
	return false
}

// end returns the last sector of a partition. Only used by partitionsOverlap. Requires non-nil Start and Size.
func (p Partition) end() int {
	if *p.SizeMiB == 0 {
		// a size of 0 means "fill available", just return the start as the end for those.
		return *p.StartMiB
	}
	return *p.StartMiB + *p.SizeMiB - 1
}

// partitionsOverlap returns true if any explicitly dimensioned partitions overlap
func (n Disk) partitionsOverlap() bool {
	for _, p := range n.Partitions {
		// Starts of 0 are placed by sgdisk into the "largest available block" at that time.
		// We aren't going to check those for overlap since we don't have the disk geometry.
		if p.StartMiB == nil || p.SizeMiB == nil || *p.StartMiB == 0 {
			continue
		}

		for _, o := range n.Partitions {
			if o.StartMiB == nil || o.SizeMiB == nil || p == o || *o.StartMiB == 0 {
				continue
			}

			// is p.StartMiB within o?
			if *p.StartMiB >= *o.StartMiB && *p.StartMiB <= o.end() {
				return true
			}

			// is p.end() within o?
			if p.end() >= *o.StartMiB && p.end() <= o.end() {
				return true
			}

			// do p.StartMiB and p.end() straddle o?
			if *p.StartMiB < *o.StartMiB && p.end() > o.end() {
				return true
			}
		}
	}
	return false
}

func (n Disk) partitionsMixZeroesAndNonexistence() bool {
	hasZero := false
	hasShouldNotExist := false
	for _, p := range n.Partitions {
		hasShouldNotExist = hasShouldNotExist || (p.ShouldExist != nil && !*p.ShouldExist)
		hasZero = hasZero || (p.Number == 0)
	}
	return hasZero && hasShouldNotExist
}
