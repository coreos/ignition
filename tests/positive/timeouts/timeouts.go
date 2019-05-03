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
	"sync"
	"time"

	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, IncreaseHTTPResponseHeadersTimeout())
	register.Register(register.PositiveTest, ConfirmHTTPBackoffWorks())
}

var (
	respondDelayServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the connection open for 11 seconds, then return
		time.Sleep(time.Second * 11)
		w.WriteHeader(http.StatusOK)
	}))

	lastResponses          = map[string]time.Time{}
	lastResponsesLock      = sync.Mutex{}
	respondThrottledServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var status int
		lastResponsesLock.Lock()
		lastResponse, ok := lastResponses[r.RequestURI]
		if ok && time.Since(lastResponse) > time.Second*4 {
			// Only respond successfully if it's been more than 4 seconds since
			// the last attempt
			status = http.StatusOK
		} else {
			status = http.StatusInternalServerError
			lastResponses[r.RequestURI] = time.Now()
		}
		lastResponsesLock.Unlock()
		w.WriteHeader(status)
	}))
)

func IncreaseHTTPResponseHeadersTimeout() types.Test {
	name := "timeouts.http"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"timeouts": {
				"httpResponseHeaders": 12
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
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func ConfirmHTTPBackoffWorks() types.Test {
	name := "timeouts.http.backoff"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version"
		},
		"storage": {
		    "files": [
			    {
					"path": "/foo/bar",
					"contents": {
						"source": "%s/$version"
					}
				}
			]
		}
	}`, respondThrottledServer.URL)
	configMinVersion := "3.0.0"
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
