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

package blackbox

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pin/tftp"
)

var (
	servedPublicKey = []byte(`-----BEGIN CERTIFICATE-----
MIICzTCCAlKgAwIBAgIJALTP0pfNBMzGMAoGCCqGSM49BAMCMIGZMQswCQYDVQQG
EwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzETMBEGA1UECgwKQ29yZU9TIEluYzEUMBIGA1UECwwLRW5naW5lZXJpbmcxEzAR
BgNVBAMMCmNvcmVvcy5jb20xHTAbBgkqhkiG9w0BCQEWDm9lbUBjb3Jlb3MuY29t
MB4XDTE4MDEyNTAwMDczOVoXDTI4MDEyMzAwMDczOVowgZkxCzAJBgNVBAYTAlVT
MRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNpc2NvMRMw
EQYDVQQKDApDb3JlT1MgSW5jMRQwEgYDVQQLDAtFbmdpbmVlcmluZzETMBEGA1UE
AwwKY29yZW9zLmNvbTEdMBsGCSqGSIb3DQEJARYOb2VtQGNvcmVvcy5jb20wdjAQ
BgcqhkjOPQIBBgUrgQQAIgNiAAQDEhfHEulYKlANw9eR5l455gwzAIQuraa049Rh
vM7PPywaiD8DobteQmE8wn7cJSzOYw6GLvrL4Q1BO5EFUXknkW50t8lfnUeHveCN
sqvm82F1NVevVoExAUhDYmMREa6jZDBiMA8GA1UdEQQIMAaHBH8AAAEwHQYDVR0O
BBYEFEbFy0SPiF1YXt+9T3Jig2rNmBtpMB8GA1UdIwQYMBaAFEbFy0SPiF1YXt+9
T3Jig2rNmBtpMA8GA1UdEwEB/wQFMAMBAf8wCgYIKoZIzj0EAwIDaQAwZgIxAOul
t3MhI02IONjTDusl2YuCxMgpy2uy0MPkEGUHnUOsxmPSG0gEBCNHyeKVeTaPUwIx
AKbyaAqbChEy9CvDgyv6qxTYU+eeBImLKS3PH2uW5etc/69V/sDojqpH3hEffsOt
9g==
-----END CERTIFICATE-----`)

	servedConfig = []byte(`{
	"ignition": { "version": "2.0.0" },
	"storage": {
		"files": [{
		  "filesystem": "root",
		  "path": "/foo/bar",
		  "contents": { "source": "data:,example%20file%0A" }
		}]
	}
}`)

	servedContents = []byte(`asdf
fdsa`)
)

// HTTP Server
func (server *HTTPServer) Config(w http.ResponseWriter, r *http.Request) {
	w.Write(servedConfig)
}

func (server *HTTPServer) Contents(w http.ResponseWriter, r *http.Request) {
	w.Write(servedContents)
}

func (server *HTTPServer) Certificates(w http.ResponseWriter, r *http.Request) {
	w.Write(servedPublicKey)
}

func errorHandler(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

// headerCheck validates that all required headers are present
func headerCheck(w http.ResponseWriter, r *http.Request) {
	if val, ok := r.Header["X-Auth"]; ok {
		if val[0] != "r8ewap98gfh4d8" {
			errorHandler(w, "X-Auth header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "X-Auth header is missing")
		return
	}

	if val, ok := r.Header["Keep-Alive"]; ok {
		if val[0] != "300" {
			errorHandler(w, "Keep-Alive header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "Keep-Alive header is missing")
		return
	}
}

func (server *HTTPServer) ConfigHeaders(w http.ResponseWriter, r *http.Request) {
	headerCheck(w, r)

	// Additional check that the system header is present in the request
	if val, ok := r.Header["Accept"]; ok {
		if val[0] != "application/vnd.coreos.ignition+json; version=2.4.0, application/vnd.coreos.ignition+json; version=1; q=0.5, */*; q=0.1" {
			errorHandler(w, "Accept header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "Accept header is missing")
		return
	}

	w.Write(servedConfig)
}

func (server *HTTPServer) ContentsHeaders(w http.ResponseWriter, r *http.Request) {
	headerCheck(w, r)

	w.Write(servedContents)
}

func (server *HTTPServer) CertificatesHeaders(w http.ResponseWriter, r *http.Request) {
	headerCheck(w, r)

	w.Write(servedPublicKey)
}

// ConfigReplaceOriginalHeaders validates that system headers were overwritten
func (server *HTTPServer) ConfigReplaceOriginalHeaders(w http.ResponseWriter, r *http.Request) {
	if val, ok := r.Header["Accept"]; ok {
		if val[0] != "text/html, application/json" {
			errorHandler(w, "Accept header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "Accept header is missing")
		return
	}

	w.Write(servedConfig)
}

// redirectedHeaderCheck validates that user's headers from the original request are missing
func redirectedHeaderCheck(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Header["X-Auth"]; ok {
		errorHandler(w, "Found redundant header X-Auth")
		return
	}

	if _, ok := r.Header["Keep-Alive"]; ok {
		errorHandler(w, "Found redundant header Keep-Alive")
		return
	}
}

// ConfigRedirect redirects the request to ConfigRedirected
func (server *HTTPServer) ConfigRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://127.0.0.1:8080/config_headers_redirected", http.StatusFound)
}

// ConfigRedirected validates the request from ConfigRedirect
func (server *HTTPServer) ConfigRedirected(w http.ResponseWriter, r *http.Request) {
	// First, check that all default headers are present
	if val, ok := r.Header["Accept-Encoding"]; ok {
		if val[0] != "identity" {
			errorHandler(w, "Accept-Encoding header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "Accept-Encoding header is missing")
		return
	}

	if val, ok := r.Header["Accept"]; ok {
		if val[0] != "application/vnd.coreos.ignition+json; version=2.4.0, application/vnd.coreos.ignition+json; version=1; q=0.5, */*; q=0.1" {
			errorHandler(w, "Accept header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "Accept header is missing")
		return
	}

	// Then check that all hedears from the original requests were erased
	redirectedHeaderCheck(w, r)

	w.Write(servedConfig)
}

// ContentsRedirect redirects the request to ContentsRedirected
func (server *HTTPServer) ContentsRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://127.0.0.1:8080/contents_headers_redirected", http.StatusFound)
}

// ContentsRedirected validates the request from ContentsRedirect
func (server *HTTPServer) ContentsRedirected(w http.ResponseWriter, r *http.Request) {
	redirectedHeaderCheck(w, r)

	w.Write(servedContents)
}

// CertificatesRedirect redirects the request to CertificatesRedirected
func (server *HTTPServer) CertificatesRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://127.0.0.1:8080/certificates_headers_redirected", http.StatusFound)
}

// CertificatesRedirected validates the request from CertificatesRedirect
func (server *HTTPServer) CertificatesRedirected(w http.ResponseWriter, r *http.Request) {
	redirectedHeaderCheck(w, r)

	w.Write(servedPublicKey)
}

type HTTPServer struct{}

func (server *HTTPServer) Start() {
	http.HandleFunc("/contents", server.Contents)
	http.HandleFunc("/contents_headers", server.ContentsHeaders)
	http.HandleFunc("/contents_headers_redirect", server.ContentsRedirect)
	http.HandleFunc("/contents_headers_redirected", server.ContentsRedirected)
	http.HandleFunc("/certificates", server.Certificates)
	http.HandleFunc("/certificates_headers", server.CertificatesHeaders)
	http.HandleFunc("/certificates_headers_redirect", server.CertificatesRedirect)
	http.HandleFunc("/certificates_headers_redirected", server.CertificatesRedirected)
	http.HandleFunc("/config", server.Config)
	http.HandleFunc("/config_headers", server.ConfigHeaders)
	http.HandleFunc("/config_headers_replace", server.ConfigReplaceOriginalHeaders)
	http.HandleFunc("/config_headers_redirect", server.ConfigRedirect)
	http.HandleFunc("/config_headers_redirected", server.ConfigRedirected)

	s := &http.Server{Addr: ":8080"}
	go s.ListenAndServe()
}

// TFTP Server
func (server *TFTPServer) ReadHandler(filename string, rf io.ReaderFrom) error {
	var buf *bytes.Reader
	if strings.Contains(filename, "contents") {
		buf = bytes.NewReader([]byte(`asdf
fdsa`))
	} else if strings.Contains(filename, "config") {
		buf = bytes.NewReader([]byte(`{
        "ignition": { "version": "2.0.0" },
        "storage": {
                "files": [{
                  "filesystem": "root",
                  "path": "/foo/bar",
                  "contents": { "source": "data:,example%20file%0A" }
                }]
        }
}`))
	} else {
		return fmt.Errorf("no such file %q", filename)
	}

	_, err := rf.ReadFrom(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	return nil
}

type TFTPServer struct{}

func (server *TFTPServer) Start() {
	s := tftp.NewServer(server.ReadHandler, nil)
	s.SetTimeout(5 * time.Second)
	go s.ListenAndServe(":69")
}
