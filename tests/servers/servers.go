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

package servers

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/ignition/v2/tests/fixtures"
	"github.com/pin/tftp"
)

var (
	servedConfig = []byte(`{
	"ignition": { "version": "3.0.0" },
	"storage": {
		"files": [{
			"path": "/foo/bar",
			"contents": { "source": "data:,example%20file%0A" }
		}]
	}
}`)
	servedContents = []byte(`asdf
fdsa`)

	// export these so tests don't have to hard-code them everywhere
	configRawHash             = sha512.Sum512(servedConfig)
	contentsRawHash           = sha512.Sum512(servedContents)
	publicKeyRawHash          = sha512.Sum512(fixtures.PublicKey)
	configRawHashForSHA256    = sha256.Sum256(servedConfig)
	contentsRawHashForSHA256  = sha256.Sum256(servedContents)
	publicKeyRawHashForSHA256 = sha256.Sum256(fixtures.PublicKey)
	ConfigHash                = hex.EncodeToString(configRawHash[:])
	ContentsHash              = hex.EncodeToString(contentsRawHash[:])
	PublicKeyHash             = hex.EncodeToString(publicKeyRawHash[:])
	ConfigHashForSHA256       = hex.EncodeToString(configRawHashForSHA256[:])
	ContentsHashForSHA256     = hex.EncodeToString(contentsRawHashForSHA256[:])
	PublicKeyHashForSHA256    = hex.EncodeToString(publicKeyRawHashForSHA256[:])
)

// HTTP Server
func (server *HTTPServer) Config(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(servedConfig)
}

func (server *HTTPServer) Contents(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(servedContents)
}

func (server *HTTPServer) Certificates(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(fixtures.PublicKey)
}

func (server *HTTPServer) CABundle(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(fixtures.CABundle)
}

func compress(contents []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(contents); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (server *HTTPServer) ConfigCompressed(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(compress(servedConfig))
}

func (server *HTTPServer) ContentsCompressed(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(compress(servedContents))
}

func (server *HTTPServer) CertificatesCompressed(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(compress(fixtures.PublicKey))
}

func errorHandler(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(message))
}

// headerCheck validates that all required headers are present
func headerCheck(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{
		"X-Auth":          "r8ewap98gfh4d8",
		"Keep-Alive":      "300",
		"Accept":          "application/vnd.coreos.ignition+json;version=3.2.0, */*;q=0.1",
		"Accept-Encoding": "identity",
	}

	for headerName, headerValue := range headers {
		if val, ok := r.Header[headerName]; ok {
			if val[0] != headerValue {
				errorHandler(w, headerName+" header value is incorrect")
				return
			}
		} else {
			errorHandler(w, headerName+" header is missing")
			return
		}
	}

	if val, ok := r.Header["User-Agent"]; ok {
		if !strings.HasPrefix(val[0], "Ignition/") {
			errorHandler(w, "User-Agent header value is incorrect")
			return
		}
	} else {
		errorHandler(w, "User-Agent header is missing")
		return
	}
}

func overwrittenHeaderCheck(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{
		"Keep-Alive":      "1000",
		"Accept":          "application/json",
		"Accept-Encoding": "identity, compress",
		"User-Agent":      "MyUA",
	}

	for headerName, headerValue := range headers {
		if val, ok := r.Header[headerName]; ok {
			if val[0] != headerValue {
				errorHandler(w, headerName+" header value is incorrect")
				return
			}
		} else {
			errorHandler(w, headerName+" header is missing")
			return
		}
	}
}

func (server *HTTPServer) ConfigHeaders(w http.ResponseWriter, r *http.Request) {
	headerCheck(w, r)

	_, _ = w.Write(servedConfig)
}

func (server *HTTPServer) ContentsHeaders(w http.ResponseWriter, r *http.Request) {
	headerCheck(w, r)

	_, _ = w.Write(servedContents)
}

func (server *HTTPServer) CertificatesHeaders(w http.ResponseWriter, r *http.Request) {
	headerCheck(w, r)

	_, _ = w.Write(fixtures.PublicKey)
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
	redirectedHeaderCheck(w, r)

	_, _ = w.Write(servedConfig)
}

// ContentsRedirect redirects the request to ContentsRedirected
func (server *HTTPServer) ContentsRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://127.0.0.1:8080/contents_headers_redirected", http.StatusFound)
}

// ContentsRedirected validates the request from ContentsRedirect
func (server *HTTPServer) ContentsRedirected(w http.ResponseWriter, r *http.Request) {
	redirectedHeaderCheck(w, r)

	_, _ = w.Write(servedContents)
}

// CertificatesRedirect redirects the request to CertificatesRedirected
func (server *HTTPServer) CertificatesRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://127.0.0.1:8080/certificates_headers_redirected", http.StatusFound)
}

// CertificatesRedirected validates the request from CertificatesRedirect
func (server *HTTPServer) CertificatesRedirected(w http.ResponseWriter, r *http.Request) {
	redirectedHeaderCheck(w, r)

	_, _ = w.Write(fixtures.PublicKey)
}

func (server *HTTPServer) ConfigHeadersOverwrite(w http.ResponseWriter, r *http.Request) {
	overwrittenHeaderCheck(w, r)

	_, _ = w.Write(servedConfig)
}

func (server *HTTPServer) ContentsHeadersOverwrite(w http.ResponseWriter, r *http.Request) {
	overwrittenHeaderCheck(w, r)

	_, _ = w.Write(servedContents)
}

func (server *HTTPServer) CertificatesHeadersOverwrite(w http.ResponseWriter, r *http.Request) {
	overwrittenHeaderCheck(w, r)

	_, _ = w.Write(fixtures.PublicKey)
}

type HTTPServer struct{}

func (server *HTTPServer) Start() {
	http.HandleFunc("/contents", server.Contents)
	http.HandleFunc("/contents_compressed", server.ContentsCompressed)
	http.HandleFunc("/contents_headers", server.ContentsHeaders)
	http.HandleFunc("/contents_headers_redirect", server.ContentsRedirect)
	http.HandleFunc("/contents_headers_redirected", server.ContentsRedirected)
	http.HandleFunc("/contents_headers_overwrite", server.ContentsHeadersOverwrite)
	http.HandleFunc("/certificates", server.Certificates)
	http.HandleFunc("/certificates_compressed", server.CertificatesCompressed)
	http.HandleFunc("/certificates_headers", server.CertificatesHeaders)
	http.HandleFunc("/certificates_headers_redirect", server.CertificatesRedirect)
	http.HandleFunc("/certificates_headers_redirected", server.CertificatesRedirected)
	http.HandleFunc("/certificates_headers_overwrite", server.CertificatesHeadersOverwrite)
	http.HandleFunc("/config", server.Config)
	http.HandleFunc("/config_compressed", server.ConfigCompressed)
	http.HandleFunc("/config_headers", server.ConfigHeaders)
	http.HandleFunc("/config_headers_redirect", server.ConfigRedirect)
	http.HandleFunc("/config_headers_redirected", server.ConfigRedirected)
	http.HandleFunc("/config_headers_overwrite", server.ConfigHeadersOverwrite)
	http.HandleFunc("/caBundle", server.CABundle)
	s := &http.Server{Addr: ":8080"}
	go func() {
		_ = s.ListenAndServe()
	}()
}

// TFTP Server
func (server *TFTPServer) ReadHandler(filename string, rf io.ReaderFrom) error {
	var buf *bytes.Reader
	if strings.Contains(filename, "contents") {
		buf = bytes.NewReader(servedContents)
	} else if strings.Contains(filename, "config") {
		buf = bytes.NewReader(servedConfig)
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
	go func() {
		_ = s.ListenAndServe(":69")
	}()
}
