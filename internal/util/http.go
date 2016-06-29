// Copyright 2016 CoreOS, Inc.
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

package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/version"
)

// HttpClient is a simple wrapper around the Go HTTP client that standardizes
// the process and logging of fetching payloads.
type HttpClient struct {
	client *http.Client
	logger *log.Logger
}

// NewHttpClient creates a new client with the given logger.
func NewHttpClient(logger *log.Logger) HttpClient {
	return HttpClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Get performs an HTTP GET on the provided URL and returns the response body,
// HTTP status code, and error (if any).
func (c HttpClient) Get(url string) ([]byte, int, error) {
	return c.GetWithHeader(url, http.Header{})
}

// GetWithHeader performs an HTTP GET on the provided URL with the provided request header
// and returns the response body, HTTP status code, and error (if any). By
// default, User-Agent and Accept are added to the header but these can be
// overridden.
func (c HttpClient) GetWithHeader(url string, header http.Header) ([]byte, int, error) {
	var body []byte
	var status int

	err := c.logger.LogOp(func() error {
		var bodyReader io.ReadCloser
		var err error

		bodyReader, status, err = c.GetReaderWithHeader(url, header)
		if err != nil {
			return err
		}
		defer bodyReader.Close()

		body, err = ioutil.ReadAll(bodyReader)

		return err
	}, "GET %q", url)

	return body, status, err
}

// GetReader performs an HTTP GET on the provided URL and returns the response body Reader,
// HTTP status code, and error (if any).
func (c HttpClient) GetReader(url string) (io.ReadCloser, int, error) {
	return c.GetReaderWithHeader(url, http.Header{})
}

// GetReaderWithHeader performs an HTTP GET on the provided URL with the provided request header
// and returns the response body Reader, HTTP status code, and error (if any). By
// default, User-Agent and Accept are added to the header but these can be
// overridden.
func (c HttpClient) GetReaderWithHeader(url string, header http.Header) (io.ReadCloser, int, error) {
	var body io.ReadCloser
	var status int

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Ignition/"+version.Raw)
	req.Header.Set("Accept", "*")
	for key, values := range header {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return body, status, err
	}

	status = resp.StatusCode
	c.logger.Debug("GET result: %s", http.StatusText(status))
	body = resp.Body

	return body, status, err
}

// FetchConfig calls FetchConfigWithHeader with an empty set of headers.
func (c HttpClient) FetchConfig(url string, acceptedStatuses ...int) []byte {
	return c.FetchConfigWithHeader(url, http.Header{}, acceptedStatuses...)
}

// FetchConfigWithHeader fetches a raw config from the provided URL and returns
// the response body on success or nil on failure. The caller must also provide
// a list of acceptable HTTP status codes and headers. If the response's status
// code is not in the provided list, it is considered a failure. The HTTP
// response must be OK, otherwise an empty (v.s. nil) config is returned. The
// provided headers are merged with a set of default headers.
func (c HttpClient) FetchConfigWithHeader(url string, header http.Header, acceptedStatuses ...int) []byte {
	var config []byte

	c.logger.LogOp(func() error {
		reqHeader := http.Header{
			"Accept-Encoding": []string{"identity"},
			"Accept":          []string{"application/vnd.coreos.ignition+json; version=2.0.0, application/vnd.coreos.ignition+json; version=1; q=0.5, */*; q=0.1"},
		}
		for key, values := range header {
			reqHeader.Del(key)
			for _, value := range values {
				reqHeader.Add(key, value)
			}
		}

		data, status, err := c.GetWithHeader(url, reqHeader)
		if err != nil {
			return err
		}

		for _, acceptedStatus := range acceptedStatuses {
			if status == acceptedStatus {
				if status == http.StatusOK {
					config = data
				} else {
					config = []byte{}
				}
				return nil
			}
		}

		return fmt.Errorf("%s", http.StatusText(status))
	}, "fetching config from %q", url)

	return config
}
