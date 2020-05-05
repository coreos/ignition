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

package files

import (
	"strings"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/servers"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, ValidateFileHashFromDataURL())
	register.Register(register.PositiveTest, ValidateFileHashFromDataURLForSHA256())
	register.Register(register.PositiveTest, ValidateFileHashFromHTTPURL())
	register.Register(register.PositiveTest, ValidateFileHashFromHTTPURLForSHA256())
	register.Register(register.PositiveTest, ValidateFileHashFromHTTPURLUsingHeaders())
}

func ValidateFileHashFromDataURL() types.Test {
	name := "files.create.withhash.dataurl"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
			"source": "data:,example%20file%0A",
			"verification": {"hash": "sha512-807e8ff949e61d23f5ee42a629ec96e9fc526b62f030cd70ba2cd5b9d97935461eacc29bf58bcd0426e9e1fdb0eda939603ed52c9c06d0712208a15cd582c60e"}
		  }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ValidateFileHashFromDataURLForSHA256() types.Test {
	name := "files.create.withhash.dataurl.sha256"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
			"source": "data:,example%20file%0A",
			"verification": {"hash": "sha256-352cb4e231c03f9941d54aeee7da755504a7f2096338c609ba5d1b82143419c6"}
		  }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
		},
	})
	configMinVersion := "3.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ValidateFileHashFromHTTPURL() types.Test {
	name := "files.create.withhash.http"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents",
			"verification": {"hash": "sha512-HASH"}
	      }
	    }]
	  }
	}`, "HASH", servers.ContentsHash, -1)
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "asdf\nfdsa",
		},
	})
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ValidateFileHashFromHTTPURLForSHA256() types.Test {
	name := "files.create.withhash.http.sha256"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents",
			"verification": {"hash": "sha256-HASH"}
	      }
	    }]
	  }
	}`, "HASH", servers.ContentsHashForSHA256, -1)
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "asdf\nfdsa",
		},
	})
	configMinVersion := "3.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ValidateFileHashFromHTTPURLUsingHeaders() types.Test {
	name := "files.create.withhash.http.headers"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
			"source": "http://127.0.0.1:8080/contents_headers",
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
			"verification": {"hash": "sha512-1a04c76c17079cd99e688ba4f1ba095b927d3fecf2b1e027af361dfeafb548f7f5f6fdd675aaa2563950db441d893ca77b0c3e965cdcb891784af96e330267d7"}
	      }
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "asdf\nfdsa",
		},
	})
	configMinVersion := "3.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
