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
		http.Error(w, "Proxy is unavailable", http.StatusServiceUnavailable)
	}))

	mockConfigURL = "http://www.fake.tld"
)

func init() {
	register.Register(register.NegativeTest, ErrorsWhenProxyIsUnavailable())
}

func ErrorsWhenProxyIsUnavailable() types.Test {
	name := "proxy.unreachable"
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
				"noProxy": [""]
			},
			"timeouts": {
				"httpResponseHeaders": 1,
				"httpTotal": 1
			}
		}
	}`, mockConfigURL, proxyServer.URL, proxyServer.URL)
	configMinVersion := "3.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
