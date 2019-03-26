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
	// export these so tests don't have to hard-code them everywhere
	configRawHash   = sha512.Sum512(servedConfig)
	contentsRawHash = sha512.Sum512(servedContents)
	ConfigHash      = hex.EncodeToString(configRawHash[:])
	ContentsHash    = hex.EncodeToString(contentsRawHash[:])
)

// HTTP Server
func (server *HTTPServer) Config(w http.ResponseWriter, r *http.Request) {
	w.Write(servedConfig)
}

func (server *HTTPServer) Contents(w http.ResponseWriter, r *http.Request) {
	w.Write(servedContents)
}

type HTTPServer struct{}

func (server *HTTPServer) Start() {
	http.HandleFunc("/contents", server.Contents)
	http.HandleFunc("/config", server.Config)

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
