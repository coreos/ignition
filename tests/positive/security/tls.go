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
	"net/http"
	"net/http/httptest"

	"github.com/coreos/ignition/v2/tests/fixtures"
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/servers"
	"github.com/coreos/ignition/v2/tests/types"

	"github.com/vincent-petithory/dataurl"
)

func init() {
	cer, err := tls.X509KeyPair(fixtures.PublicKey, fixtures.PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("error loading x509 keypair: %v", err))
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	customCAServer.TLS = config
	customCAServer.StartTLS()

	cer2, err := tls.X509KeyPair(fixtures.PublicKey2, fixtures.PrivateKey2)
	if err != nil {
		panic(fmt.Sprintf("error loading x509 keypair2: %v", err))
	}
	config2 := &tls.Config{Certificates: []tls.Certificate{cer2}}
	customCAServer2.TLS = config2
	customCAServer2.StartTLS()

	register.Register(register.PositiveTest, AppendConfigCustomCert())
	register.Register(register.PositiveTest, FetchFileCustomCert())
	register.Register(register.PositiveTest, FetchFileCABundleCert())
	register.Register(register.PositiveTest, FetchFileCustomCertHTTP())
	register.Register(register.PositiveTest, FetchFileCABundleCertHTTP())
	register.Register(register.PositiveTest, FetchFileCustomCertHTTPCompressed())
	register.Register(register.PositiveTest, FetchFileCustomCertHTTPUsingHeaders())
	register.Register(register.PositiveTest, FetchFileCustomCertHTTPUsingHeadersWithRedirect())
	register.Register(register.PositiveTest, FetchFileCustomCertHTTPUsingOverwrittenHeaders())
}

var (
	customCAServerFile = []byte(`{
			"ignition": { "version": "3.0.0" },
			"storage": {
				"files": [{
					"path": "/foo/bar",
					"contents": { "source": "data:,example%20file%0A" }
				}]
			}
		}`)

	customCAServerFile2 = []byte(`{
			"ignition": { "version": "3.0.0" },
			"storage": {
				"files": [{
					"path": "/foo/bar2",
					"contents": { "source": "data:,example%20file2%0A" }
				}]
			}
		}`)

	customCAServer = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(customCAServerFile)
	}))
	customCAServer2 = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(customCAServerFile2)
	}))
)

func AppendConfigCustomCert() types.Test {
	name := "tls.appendconfig"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"config": {
			  "merge": [{
				"source": %q
			  }]
			},
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"source": %q
					}]
				}
			}
		}
	}`, customCAServer.URL, dataurl.EncodeBytes(fixtures.PublicKey))
	configMinVersion := "3.0.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Contents: "example file\n",
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

func FetchFileCustomCert() types.Test {
	name := "tls.fetchfile"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"source": %q
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, dataurl.EncodeBytes(fixtures.PublicKey), customCAServer.URL)
	configMinVersion := "3.0.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
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

func FetchFileCABundleCert() types.Test {
	name := "tls.fetchfile.cabundle"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"source": %q
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			},
			{
				"path": "/foo/bar2",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, dataurl.EncodeBytes(fixtures.CABundle), customCAServer.URL, customCAServer2.URL)
	configMinVersion := "3.0.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
		},
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar2",
			},
			Contents: string(customCAServerFile2),
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
						"source": "http://127.0.0.1:8080/certificates"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "3.0.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
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
						"source": "http://127.0.0.1:8080/caBundle"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			},
			{
				"path": "/foo/bar2",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL, customCAServer2.URL)
	configMinVersion := "3.0.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
		},
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar2",
			},
			Contents: string(customCAServerFile2),
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

func FetchFileCustomCertHTTPCompressed() types.Test {
	name := "tls.fetchfile.http.compressed"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"compression": "gzip",
						"source": "http://127.0.0.1:8080/certificates_compressed",
						"verification": {
							"hash": "sha512-%v"
						}
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, servers.PublicKeyHash, customCAServer.URL)
	configMinVersion := "3.1.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
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

func FetchFileCustomCertHTTPUsingHeaders() types.Test {
	name := "tls.fetchfile.http.headers"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
						"source": "http://127.0.0.1:8080/certificates_headers"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "3.1.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
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

func FetchFileCustomCertHTTPUsingHeadersWithRedirect() types.Test {
	name := "tls.fetchfile.http.headers.redirect"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"httpHeaders": [{"name": "X-Auth", "value": "r8ewap98gfh4d8"}, {"name": "Keep-Alive", "value": "300"}],
						"source": "http://127.0.0.1:8080/certificates_headers_redirect"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "3.1.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
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

func FetchFileCustomCertHTTPUsingOverwrittenHeaders() types.Test {
	name := "tls.fetchfile.http.headers.overwrite"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := fmt.Sprintf(`{
		"ignition": {
			"version": "$version",
			"security": {
				"tls": {
					"certificateAuthorities": [{
						"httpHeaders": [
							{"name": "Keep-Alive", "value": "1000"},
							{"name": "Accept", "value": "application/json"},
							{"name": "Accept-Encoding", "value": "identity, compress"},
							{"name": "User-Agent", "value": "MyUA"}
						],
						"source": "http://127.0.0.1:8080/certificates_headers_overwrite"
					}]
				}
			}
		},
		"storage": {
			"files": [{
				"path": "/foo/bar",
				"contents": {
					"source": %q
				}
			}]
		}
	}`, customCAServer.URL)
	configMinVersion := "3.1.0"

	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Contents: string(customCAServerFile),
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
