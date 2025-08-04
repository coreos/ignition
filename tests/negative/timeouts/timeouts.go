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

package timeouts

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.NegativeTest, DecreaseHTTPResponseHeadersTimeout())
	register.Register(register.NegativeTest, AppendWithHTTPTimeouts())
	register.Register(register.NegativeTest, AppendLowerHTTPTimeouts())
	register.Register(register.NegativeTest, AppendNoneThenLowerHTTPTimeouts())
}

var (
	respondDelayServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the connection open for 2 seconds, then return
		time.Sleep(time.Second * 2)
		w.WriteHeader(http.StatusOK)
	}))

	configDelayServerURL string
	configDelayServer    = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the connection open for 2 seconds, then return
		time.Sleep(time.Second * 2)
		// Give a config that merges ourselves and sets the timeouts to 1
		// second (less than we wait!)
		_, _ = w.Write([]byte(fmt.Sprintf(`{
			"ignition": {
				"version": "$version",
				"config": {
					"merge": [{
						"source": %q
					}]
				},
				"timeouts": {
					"httpResponseHeaders": 1,
					"httpTotal": 1
				}
			}
		}`, configDelayServerURL)))
	}))
)

func DecreaseHTTPResponseHeadersTimeout() types.Test {
	name := "timeouts.file.create.http.timesout"
	in := types.GetBaseDisk()
	out := in
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"timeouts": {
				"httpResponseHeaders": 1,
				"httpTotal": 10
			}
		},
		"storage": {
		    "files": [
			    {
					"path": "/foo/bar",
					"contents": {
						"source": %q
					}
				}
			]
		}
	}`, respondDelayServer.URL)
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendWithHTTPTimeouts() types.Test {
	name := "timeouts.config.merge.timesout"
	in := types.GetBaseDisk()
	out := in
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
				"merge": [{
					"source": %q
				}]
			},
			"timeouts": {
				"httpResponseHeaders": 1,
				"httpTotal": 1
			}
		}
	}`, configDelayServer.URL)
	configDelayServerURL = configDelayServer.URL
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendLowerHTTPTimeouts() types.Test {
	name := "timeouts.config.mergeslowertimeout"
	in := types.GetBaseDisk()
	out := in
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
				"merge": [{
					"source": %q
				}]
			}
		}
	}`, configDelayServer.URL)
	configDelayServerURL = configDelayServer.URL
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func AppendNoneThenLowerHTTPTimeouts() types.Test {
	// If an initial config specifies timeouts, and then merges a config with
	// no timeouts, the initial timeouts should still apply

	var (
		emptyConfigDelayServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Hold the connection open for 2 seconds, then return
			time.Sleep(time.Second * 2)
			// Give a config that merges ourselves and sets the timeouts to 1
			// second (less than we wait!)
			_, _ = w.Write([]byte(`{
				"ignition": {
					"version": "$version"
				}
			}`))
		}))

		configNoDelayServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Give a config that merges ourselves and sets the timeouts to 1
			// second (less than we wait!)
			_, _ = w.Write([]byte(fmt.Sprintf(`{
			"ignition": 
				"version": "$version",
				"config": {
					"merge": [{
						"source": %q
					}]
				}
			}
		}`, emptyConfigDelayServer.URL)))
		}))
	)

	name := "timeouts.config.merge.nonethenlower"
	in := types.GetBaseDisk()
	out := in
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
				"merge": [{
					"source": %q
				}]
			},
			"timeouts": {
				"httpResponseHeaders": 1,
				"httpTotal": 1
			}
		}
	}`, configNoDelayServer.URL)
	configMinVersion := "3.0.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
