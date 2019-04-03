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

package general

import (
	"fmt"

	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"

	"github.com/vincent-petithory/dataurl"
)

func init() {
	register.Register(register.NegativeTest, ReplaceConfigWithInvalidHash())
	register.Register(register.NegativeTest, AppendConfigWithInvalidHash())
	register.Register(register.NegativeTest, ReplaceConfigWithMissingFileHTTP())
	register.Register(register.NegativeTest, ReplaceConfigWithMissingFileTFTP())
	register.Register(register.NegativeTest, AppendConfigWithMissingFileHTTP())
	register.Register(register.NegativeTest, AppendConfigWithMissingFileTFTP())
	register.Register(register.NegativeTest, VersionOnlyConfig24())
	register.Register(register.NegativeTest, VersionOnlyConfig32())
	register.Register(register.NegativeTest, MergingCanFail())
}

func ReplaceConfigWithInvalidHash() types.Test {
	name := "Replace Config with Invalid Hash"
	in := types.GetBaseDisk()
	out := in
	mntDevices := []types.MntDevice{
		{
			Label:        "EFI-SYSTEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
	        "source": "http://127.0.0.1:8080/config",
			"verification": { "hash": "sha512-1a04c76c17079cd99e688ba4f1ba095b927d3fecf2b1e027af361dfeafb548f7f5f6fdd675aaa2563950db441d893ca77b0c3e965cdcb891784af96e330267d7" }
	      }
	    }
	  }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendConfigWithInvalidHash() types.Test {
	name := "Append Config with Invalid Hash"
	in := types.GetBaseDisk()
	out := in
	mntDevices := []types.MntDevice{
		{
			Label:        "EFI-SYSTEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
	        "source": "http://127.0.0.1:8080/config",
			"verification": { "hash": "sha512-1a04c76c17079cd99e688ba4f1ba095b927d3fecf2b1e027af361dfeafb548f7f5f6fdd675aaa2563950db441d893ca77b0c3e965cdcb891784af96e330267d7" }
	      }]
	    }
	  },
      "storage": {
        "files": [{
          "path": "/foo/bar2",
          "contents": { "source": "data:,another%20example%20file%0A" }
        }]
      }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		MntDevices:       mntDevices,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReplaceConfigWithMissingFileHTTP() types.Test {
	name := "Replace Config with Missing File - HTTP"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
	        "source": "http://127.0.0.1:8080/asdf"
	      }
	    }
	  }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ReplaceConfigWithMissingFileTFTP() types.Test {
	name := "Replace Config with Missing File - TFTP"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "replace": {
	        "source": "tftp://127.0.0.1:69/asdf"
	      }
	    }
	  }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendConfigWithMissingFileHTTP() types.Test {
	name := "Append Config with Missing File - HTTP"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
	        "source": "http://127.0.0.1:8080/asdf"
	      }]
	    }
	  }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendConfigWithMissingFileTFTP() types.Test {
	name := "Append Config with Missing File - TFTP"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": {
	    "version": "$version",
	    "config": {
	      "merge": [{
	        "source": "tftp://127.0.0.1:69/asdf"
	      }]
	    }
	  }
	}`
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func VersionOnlyConfig24() types.Test {
	name := "Version Only Config 2.4.0-experimental"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": {
	    "version": "2.4.0-experimental"
	  }
	}`

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
	}
}

func VersionOnlyConfig32() types.Test {
	name := "Version Only Config 3.2.0-experimental"
	in := types.GetBaseDisk()
	out := in
	config := `{
	  "ignition": {
	    "version": "3.2.0-experimental"
	  }
	}`

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		ConfigShouldBeBad: true,
	}
}

func MergingCanFail() types.Test {
	name := "Merging Can Fail"
	configMinVersion := "3.0.0-experimental"
	in := types.GetBaseDisk()
	out := in
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "DEVICE", // no $, since it'll get mangled by the url encoding
		},
	}
	appendedConfig := `{
	  "ignition": {
	    "version": "3.0.0"
	  },
	  "storage": {
	    "filesystems": [{
	      "format": "",
	      "device": "DEVICE"
	    }]
	  }
	}`
	du := dataurl.New([]byte(appendedConfig), "text/plain")
	du.Encoding = dataurl.EncodingASCII // needed to make sure $DEVICE gets decoded correctly

	config := fmt.Sprintf(`{
	  "ignition": {
	    "version": "3.0.0",
	    "config": {
	      "merge": [{
	        "source": "%s"
	      }]
	    }
	  },
	  "storage": {
	    "filesystems": [{
	      "path": "/foo",
	      "format": "ext4",
	      "device": "DEVICE"
	    }]
	  }
	}`, du.String())

	return types.Test{
		Name:              name,
		In:                in,
		Out:               out,
		Config:            config,
		MntDevices:        mntDevices,
		ConfigShouldBeBad: false,
		ConfigMinVersion:  configMinVersion,
	}
}
