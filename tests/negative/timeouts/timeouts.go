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

	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.NegativeTest, DecreaseHTTPResponseHeadersTimeout())
}

var (
	respondDelayServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the connection open for 2 seconds, then return
		time.Sleep(time.Second * 2)
		w.WriteHeader(http.StatusOK)
	}))
)

func DecreaseHTTPResponseHeadersTimeout() types.Test {
	name := "Decrease HTTP Response Headers Timeout"
	in := types.GetBaseDisk()
	out := in
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "2.1.0",
			"timeouts": {
				"httpResponseHeaders": 1,
				"httpTotal": 10,
			}
		},
		"storage": {
		    "files": [
			    {
					"filesystem": "root",
					"path": "/foo/bar",
					"contents": {
						"source": %q
					}
				}
			]
		}
	}`, respondDelayServer.URL)

	return types.Test{
		Name:   name,
		In:     in,
		Out:    out,
		Config: config,
	}
}
