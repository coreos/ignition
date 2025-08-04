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
	register.Register(register.PositiveTest, CreateFileFromRemoteContentsHTTP())
	register.Register(register.PositiveTest, CreateFileFromRemoteContentsHTTPCompressed())
	register.Register(register.PositiveTest, CreateFileFromRemoteContentsHTTPUsingHeaders())
	register.Register(register.PositiveTest, CreateFileFromRemoteContentsHTTPUsingHeadersWithRedirect())
	register.Register(register.PositiveTest, CreateFileFromRemoteContentsHTTPUsingOverwrittenHeaders())
	register.Register(register.PositiveTest, CreateFileFromRemoteContentsTFTP())
}

func CreateFileFromRemoteContentsHTTP() types.Test {
	name := "files.create.http"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
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
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func CreateFileFromRemoteContentsHTTPCompressed() types.Test {
	name := "files.create.http.compressed"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := strings.Replace(`{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
	        "compression": "gzip",
	        "source": "http://127.0.0.1:8080/contents_compressed",
	        "verification": {
	          "hash": "sha512-HASH"
	        }
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
	configMinVersion := "3.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func CreateFileFromRemoteContentsHTTPUsingHeaders() types.Test {
	name := "files.create.http.headers"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
	        "source": "http://127.0.0.1:8080/contents_headers"
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

func CreateFileFromRemoteContentsHTTPUsingHeadersWithRedirect() types.Test {
	name := "files.create.http.headers.redirect"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
			"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
	        "source": "http://127.0.0.1:8080/contents_headers_redirect"
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

func CreateFileFromRemoteContentsHTTPUsingOverwrittenHeaders() types.Test {
	name := "files.create.http.headers.overwrite"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
	  "ignition": { "version": "$version" },
	  "storage": {
	    "files": [{
	      "path": "/foo/bar",
	      "contents": {
			"httpHeaders": [
				{"name": "Keep-Alive", "value": "1000"},
				{"name": "Accept", "value": "application/json"},
				{"name": "Accept-Encoding", "value": "identity, compress"},
				{"name": "User-Agent", "value": "MyUA"}
			],
	        "source": "http://127.0.0.1:8080/contents_headers_overwrite"
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

func CreateFileFromRemoteContentsTFTP() types.Test {
	name := "files.create.tftp"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
          "ignition": { "version": "$version" },
          "storage": {
            "files": [{
              "path": "/foo/bar",
              "contents": {
                "source": "tftp://127.0.0.1:69/contents"
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
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
