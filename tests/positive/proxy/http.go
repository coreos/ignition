// Copyright 2019 Red Hat, Inc.
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

package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

var (
	proxyServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(ignConfig))
	}))

	mockConfigURL = "http://www.fake.tld"

	ignConfig = `{
		"ignition": { "version": "3.1.0" },
		"storage": {
			"files": [{
				"path": "/bar/foo",
				"contents": { "source": "data:,example%20file%20proxy%0A" }
			}]
		}
	}`
)

func init() {
	register.Register(register.PositiveTest, CanUseProxyForRetrievingConfig())
	register.Register(register.PositiveTest, CanUseNoProxyForRetrievingConfig())
}

func CanUseProxyForRetrievingConfig() types.Test {
	name := "proxy.getconfig"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
				"replace": {
					"source": "%s/"
				}
			},
			"proxy": {
				"httpProxy": "%s",
				"httpsProxy": "%s",
				"noProxy": ["localhost", "127.0.0.1"]
			}
		}
	}`, mockConfigURL, proxyServer.URL, proxyServer.URL)
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "foo",
				Directory: "bar",
			},
			Contents: "example file proxy\n",
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

func CanUseNoProxyForRetrievingConfig() types.Test {
	name := "proxy.getconfig.noproxy"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
				"replace": {
					"source": "%s"
				}
			},
			"proxy": {
				"httpProxy": "%s",
				"httpsProxy": "%s",
				"noProxy": ["localhost", "127.0.0.1"]
			}
		}
	}`, "http://127.0.0.1:8080/config", proxyServer.URL, proxyServer.URL)
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
