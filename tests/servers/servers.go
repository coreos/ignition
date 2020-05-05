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
	servedCABundle = []byte(`-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
# CustomCAServer1 certificate
-----BEGIN CERTIFICATE-----
MIICrDCCAjOgAwIBAgIUbFS1ugcEYYGQoTiV6O//r3wdO58wCgYIKoZIzj0EAwIw
gYQxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJOQzEQMA4GA1UEBwwHUmFsZWlnaDEQ
MA4GA1UECgwHUmVkIEhhdDEUMBIGA1UECwwLRW5naW5lZXJpbmcxDzANBgNVBAMM
BkNvcmVPUzEdMBsGCSqGSIb3DQEJARYOb2VtQGNvcmVvcy5jb20wHhcNMjAwNTA3
MjIzMzA3WhcNMzAwNTA1MjIzMzA3WjCBhDELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
Ak5DMRAwDgYDVQQHDAdSYWxlaWdoMRAwDgYDVQQKDAdSZWQgSGF0MRQwEgYDVQQL
DAtFbmdpbmVlcmluZzEPMA0GA1UEAwwGQ29yZU9TMR0wGwYJKoZIhvcNAQkBFg5v
ZW1AY29yZW9zLmNvbTB2MBAGByqGSM49AgEGBSuBBAAiA2IABD8nQPQLeHZrnklT
f8QA17V3d7b+Z9yhcao/yyiIV1ie1EgLj5Xh2dEm4dFH4UdH/fnfUQh/ZZeyggmv
8hGFH+PS0B5eln74b4S9qihd/uBKGeFFgozi+I/Hj4OFI+OarKNkMGIwDwYDVR0R
BAgwBocEfwAAATAdBgNVHQ4EFgQUovVgWNFFPhrF7XzaRteDnpfPXxowHwYDVR0j
BBgwFoAUovVgWNFFPhrF7XzaRteDnpfPXxowDwYDVR0TAQH/BAUwAwEB/zAKBggq
hkjOPQQDAgNnADBkAjBvCIr9k43oR18Z4HLTzaRfzacFzo75Lt5n0pk3PA5CrUg3
sXU6o4IxyLNFHzIJn7cCMGTMVKEzoSZDclxkEgu53WM7PQljHgL9FJScEt4hzO2u
FFNjhq0ODV1LNc1i8pQCAg==
-----END CERTIFICATE-----`)

	// export these so tests don't have to hard-code them everywhere
	configRawHash             = sha512.Sum512(servedConfig)
	contentsRawHash           = sha512.Sum512(servedContents)
	publicKeyRawHash          = sha512.Sum512(servedPublicKey)
	configRawHashForSHA256    = sha256.Sum256(servedConfig)
	contentsRawHashForSHA256  = sha256.Sum256(servedContents)
	publicKeyRawHashForSHA256 = sha256.Sum256(servedPublicKey)
	ConfigHash                = hex.EncodeToString(configRawHash[:])
	ContentsHash              = hex.EncodeToString(contentsRawHash[:])
	PublicKeyHash             = hex.EncodeToString(publicKeyRawHash[:])
	ConfigHashForSHA256       = hex.EncodeToString(configRawHashForSHA256[:])
	ContentsHashForSHA256     = hex.EncodeToString(contentsRawHashForSHA256[:])
	PublicKeyHashForSHA256    = hex.EncodeToString(publicKeyRawHashForSHA256[:])
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

func (server *HTTPServer) CABundle(w http.ResponseWriter, r *http.Request) {
	w.Write(servedCABundle)
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
	w.Write(compress(servedConfig))
}

func (server *HTTPServer) ContentsCompressed(w http.ResponseWriter, r *http.Request) {
	w.Write(compress(servedContents))
}

func (server *HTTPServer) CertificatesCompressed(w http.ResponseWriter, r *http.Request) {
	w.Write(compress(servedPublicKey))
}

func errorHandler(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

// headerCheck validates that all required headers are present
func headerCheck(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{
		"X-Auth":          "r8ewap98gfh4d8",
		"Keep-Alive":      "300",
		"Accept":          "application/vnd.coreos.ignition+json;version=3.1.0, */*;q=0.1",
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

func (server *HTTPServer) ConfigHeadersOverwrite(w http.ResponseWriter, r *http.Request) {
	overwrittenHeaderCheck(w, r)

	w.Write(servedConfig)
}

func (server *HTTPServer) ContentsHeadersOverwrite(w http.ResponseWriter, r *http.Request) {
	overwrittenHeaderCheck(w, r)

	w.Write(servedContents)
}

func (server *HTTPServer) CertificatesHeadersOverwrite(w http.ResponseWriter, r *http.Request) {
	overwrittenHeaderCheck(w, r)

	w.Write(servedPublicKey)
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
	go s.ListenAndServe()
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
	go s.ListenAndServe(":69")
}
