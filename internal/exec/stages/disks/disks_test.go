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

package disks

import (
	"fmt"
	"testing"

	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/internal/util"

	"github.com/stretchr/testify/assert"
)

func TestPartitionMatches(t *testing.T) {
	type in struct {
		info types.Partition
		spec types.Partition
	}
	type test struct {
		in  in
		out error
	}
	// test that
	// empty is ok, empty spec accepts everything, exact spec match works
	// each field can cause a problem
	baseSpec := types.Partition{
		Label:    util.StringToPtr("FOOBAR"),
		GUID:     "FOOBAR",
		TypeGUID: "FOOBAR",
		Start:    util.IntToPtr(1),
		Size:     util.IntToPtr(1),
	}

	tests := []test{
		{
			in: in{
				info: types.Partition{},
				spec: types.Partition{},
			},
			out: nil,
		},
		{
			in: in{
				info: baseSpec,
				spec: types.Partition{},
			},
			out: nil,
		},
		{
			in: in{
				info: baseSpec,
				spec: baseSpec,
			},
			out: nil,
		},
		{
			in: in{
				info: types.Partition{
					Label:    util.StringToPtr("diff"),
					GUID:     "FOOBAR",
					TypeGUID: "FOOBAR",
					Start:    util.IntToPtr(1),
					Size:     util.IntToPtr(1),
				},
				spec: baseSpec,
			},
			out: fmt.Errorf("label did not match (specified %q, got %q)", "FOOBAR", "diff"),
		},
		{
			in: in{
				info: types.Partition{
					Label:    util.StringToPtr("FOOBAR"),
					GUID:     "diff",
					TypeGUID: "FOOBAR",
					Start:    util.IntToPtr(1),
					Size:     util.IntToPtr(1),
				},
				spec: baseSpec,
			},
			out: fmt.Errorf("GUID did not match (specified %q, got %q)", "FOOBAR", "diff"),
		},
		{
			in: in{
				info: types.Partition{
					Label:    util.StringToPtr("FOOBAR"),
					GUID:     "FOOBAR",
					TypeGUID: "diff",
					Start:    util.IntToPtr(1),
					Size:     util.IntToPtr(1),
				},
				spec: baseSpec,
			},
			out: fmt.Errorf("type GUID did not match (specified %q, got %q)", "FOOBAR", "diff"),
		},
		{
			in: in{
				info: types.Partition{
					Label:    util.StringToPtr("FOOBAR"),
					GUID:     "FOOBAR",
					TypeGUID: "FOOBAR",
					Start:    util.IntToPtr(2),
					Size:     util.IntToPtr(1),
				},
				spec: baseSpec,
			},
			out: fmt.Errorf("starting sector did not match (specified %q, got %q)", 1, 2),
		},
		{
			in: in{
				info: types.Partition{
					Label:    util.StringToPtr("FOOBAR"),
					GUID:     "FOOBAR",
					TypeGUID: "FOOBAR",
					Start:    util.IntToPtr(1),
					Size:     util.IntToPtr(2),
				},
				spec: baseSpec,
			},
			out: fmt.Errorf("size did not match (specified %q, got %q)", 1, 2),
		},
	}

	for i, test := range tests {
		err := partitionMatches(test.in.info, test.in.spec)
		assert.Equal(t, test.out, err, "#%d Error did not match", i)
	}
}

func TestParseSgdiskPretend(t *testing.T) {
	type in struct {
		sgdiskOut string
		parts     []int
	}

	type out struct {
		parts map[int]sgdiskOutput
		err   error
	}

	type test struct {
		in  in
		out out
	}

	tests := []test{
		{
			in: in{
				sgdiskOut: "",
				parts:     []int{},
			},
			out: out{
				parts: nil,
				err:   nil,
			},
		},
		{
			in: in{
				sgdiskOut: `
creating new GPT entries.
Partition GUID code: 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem)
Partition unique GUID: D800A202-B471-43B3-96EB-3EA7C21EACDD
First sector: 2048 (at 1024.0 KiB)
Last sector: 264191 (at 129.0 MiB)
Partition size: 262144 sectors (128.0 MiB)
Attribute flags: 0000000000000000
Partition name: ''
The operation has completed successfully.`,
				parts: []int{1},
			},
			out: out{
				parts: map[int]sgdiskOutput{
					1: {
						start: 2048,
						end:   264191,
					},
				},
				err: nil,
			},
		},
		{
			in: in{
				sgdiskOut: `
GPT data structures destroyed! You may now partition the disk using fdisk or
other utilities.
Partition GUID code: 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem)
Partition unique GUID: D68447C3-37A4-41CF-96A6-FDDE9D2DC262
First sector: 2048 (at 1024.0 KiB)
Last sector: 264191 (at 129.0 MiB)
Partition size: 262144 sectors (128.0 MiB)
Attribute flags: 0000000000000000
Partition name: ''
Partition GUID code: 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem)
Partition unique GUID: AF5291D6-D687-41B0-AAB3-146D2486A268
First sector: 526336 (at 257.0 MiB)
Last sector: 788479 (at 385.0 MiB)
Partition size: 262144 sectors (128.0 MiB)
Attribute flags: 0000000000000000
Partition name: ''
The operation has completed successfully.`,
				parts: []int{1, 3},
			},
			out: out{
				parts: map[int]sgdiskOutput{
					1: {
						start: 2048,
						end:   264191,
					},
					3: {
						start: 526336,
						end:   788479,
					},
				},
				err: nil,
			},
		},
		{
			in: in{
				sgdiskOut: `
creating new GPT entries.
Partition GUID code: 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem)
Partition unique GUID: D800A202-B471-43B3-96EB-3EA7C21EACDD
First sector: 2048 (at 1024.0 KiB)
Last sector: 264191 (at 129.0 MiB)
Partition size: 262144 sectors (128.0 MiB)
Attribute flags: 0000000000000000
Partition name: ''
First sector: 2048 (at 1024.0 KiB)
The operation has completed successfully.`,
				parts: []int{1},
			},
			out: out{
				parts: nil,
				err:   ErrBadSgdiskOutput,
			},
		},
		{
			in: in{
				sgdiskOut: `
creating new GPT entries.
Partition GUID code: 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem)
Partition unique GUID: D800A202-B471-43B3-96EB-3EA7C21EACDD
First sector: 2048 (at 1024.0 KiB)
Partition size: 262144 sectors (128.0 MiB)
Attribute flags: 0000000000000000
Partition name: ''
The operation has completed successfully.`,
				parts: []int{1},
			},
			out: out{
				parts: nil,
				err:   ErrBadSgdiskOutput,
			},
		},
		{
			in: in{
				sgdiskOut: `
creating new GPT entries.
Partition GUID code: 0FC63DAF-8483-4772-8E79-3D69D8477DE4 (Linux filesystem)
Partition unique GUID: D800A202-B471-43B3-96EB-3EA7C21EACDD
Last sector: 2048 (at 1024.0 KiB)
Partition size: 262144 sectors (128.0 MiB)
Attribute flags: 0000000000000000
Partition name: ''
The operation has completed successfully.`,
				parts: []int{1},
			},
			out: out{
				parts: nil,
				err:   ErrBadSgdiskOutput,
			},
		},
	}

	for i, test := range tests {
		out, err := parseSgdiskPretend(test.in.sgdiskOut, test.in.parts)
		assert.Equal(t, test.out.err, err, "#%d Error did not match", i)
		assert.Equal(t, test.out.parts, out, "#%d Partitions did not match", i)
	}
}
