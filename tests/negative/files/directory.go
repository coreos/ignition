// Copyright 2022 Red Hat, Inc.
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
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.NegativeTest, DirectoryFromArchiveMustSetOverwrite())
	register.Register(register.NegativeTest, DirectoryFromArchiveMustSetArchive())
	register.Register(register.NegativeTest, DirectoryFromArchiveConflicts())
}

func DirectoryFromArchiveMustSetOverwrite() types.Test {
	name := "directories.must-set-overwrite"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [
	      {
	        "path": "/foo/bar",
	        "contents": {
	          "archive": "tar",
	          "source": "data:;base64,"
	        }
	      }
	    ]
	  }
	}`
	configMinVersion := "3.4.0-experimental"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		ConfigShouldBeBad: true,
	}
}

func DirectoryFromArchiveMustSetArchive() types.Test {
	name := "directories.must-set-archive"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [
	      {
	        "path": "/foo/bar",
	        "overwrite": true,
	        "contents": {
	          "source": "data:;base64,"
	        }
	      }
	    ]
	  }
	}`
	configMinVersion := "3.4.0-experimental"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		ConfigShouldBeBad: true,
	}
}

func DirectoryFromArchiveConflicts() types.Test {
	name := "directories.conflict"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "directories": [
	      {
	        "path": "/foo",
	        "overwrite": true,
	        "contents": {
	          "archive": "tar",
	          "source": "data:;base64,"
	        }
	      },
	      {
	        "path": "/foo/bar"
	      }
	    ],
	    "files": [
	      {
	        "path": "/foo/baz"
	      }
	    ],
	    "links": [
	      {
	        "path": "/foo/quxx",
	        "target": "/"
	      }
	    ]
	  }
	}`
	configMinVersion := "3.4.0-experimental"

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigMinVersion:  configMinVersion,
		ConfigShouldBeBad: true,
	}
}
