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

package security

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/coreos/ignition/tests/fixtures"
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	cer, err := tls.X509KeyPair(fixtures.PublicKey, fixtures.PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("error loading x509 keypair: %v", err))
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	customCAServer.TLS = config
	customCAServer.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	customCAServer.StartTLS()

	cer2, err := tls.X509KeyPair(fixtures.PublicKey2, fixtures.PrivateKey2)
	if err != nil {
		panic(fmt.Sprintf("error loading x509 keypair2: %v", err))
	}
	config2 := &tls.Config{Certificates: []tls.Certificate{cer2}}
	customCAServer2.TLS = config2
	customCAServer2.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	customCAServer2.StartTLS()

	register.Register(register.NegativeTest, AppendConfigCustomCert())
	register.Register(register.NegativeTest, FetchFileCustomCert())
	register.Register(register.NegativeTest, FetchFileCABundleCertHTTP())
	register.Register(register.NegativeTest, FetchFileCustomCertHTTP())
	register.Register(register.NegativeTest, FetchFileCustomCertInvalidHeaderHTTP())
}

var (
	customCAServerFile = []byte(`{
			"ignition": { "version": "2.0.0" },
			"storage": {
				"files": [{
					"filesystem": "root",
					"path": "/foo/bar",
					"contents": { "source": "data:,example%20file%0A" }
				}]
			}
		}`)

	customCAServerFile2 = []byte(`{
			"ignition": { "version": "2.0.0" },
			"storage": {
				"files": [{
					"filesystem": "root",
					"path": "/foo/bar2",
					"contents": { "source": "data:,example%20file2%0A" }
				}]
			}
		}`)
	customCAServer = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(customCAServerFile)
	}))
	customCAServer2 = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(customCAServerFile2)
	}))
)

func AppendConfigCustomCert() types.Test {
	name := "tls.config.merge.needsca"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
			  "append": [{
				"source": %q
			  }]
			},
			"timeouts": {
				"httpTotal": 5
			}
		}
	}`, customCAServer.URL)
	configMinVersion := "2.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func FetchFileCustomCert() types.Test {
	name := "tls.file.create.needsca"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"timeouts": {
				"httpTotal": 5
			}
		},
		"storage": {
			"files": [{
				"filesystem": "root",
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "2.1.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

// FetchFileCABundleCertHTTP fetches the ignition configs hosted
// on the TLS servers using a CA bundle that includes only the first
// server's CA key.
func FetchFileCABundleCertHTTP() types.Test {
	name := "tls.fetchfile.http.cabundle"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"source": "http://127.0.0.1:8080/certificates"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"filesystem": "root",
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			},{
				"filesystem": "root",
				"path": "/foo/bar2",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL, customCAServer2.URL)
	configMinVersion := "2.2.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func FetchFileCustomCertHTTP() types.Test {
	name := "tls.fetchfile.http"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"source": "http://127.0.0.1:8080/asdf"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"filesystem": "root",
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "2.2.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

func FetchFileCustomCertInvalidHeaderHTTP() types.Test {
	name := "tls.fetchfile.http.invalidheader"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"httpHeaders": [{"name": "X-Auth", "value": "INVALID"}, {"name": "Keep-Alive", "value": "300"}],
						"source": "http://127.0.0.1:8080/certificates_headers"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"filesystem": "root",
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "2.4.0"

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
