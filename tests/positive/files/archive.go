// Copyright 2018 CoreOS, Inc.
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

package files

import (
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ExtractTarOnRoot())
}

func ExtractTarOnRoot() types.Test {
	name := "ExtractTarOnRoot"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "2.3.0-experimental" },
	  "storage": {
	    "archives": [{
	      "filesystem": "root",
	      "path": "/opt/test",
		  "format": "tar",
		  "contents": { "source": "http://127.0.0.1:8080/tarfile" }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "/opt/test",
				Name:      "testfile",
			},
			Contents: "this is a test\n",
			Mode:     0664,
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Directory: "/opt/test",
				Name:      "testdir",
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "/opt/test",
				Name:      "testhardlink",
			},
			Target: "/opt/test/testfile",
			Hard:   true,
		},
		{
			Node: types.Node{
				Directory: "/opt/test/testdir",
				Name:      "testsymlink",
			},
			Target: "../testfile",
			Hard:   false,
		},
	})

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}
