// Copyright 2026 Red Hat, Inc.
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

package openstack

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
)

const testUserdata = `{"ignition": {"version": "3.0.0"}}`

type metadataServer struct {
	// number of meta_data.json requests answered with 404 before the
	// instance becomes known to the metadata service
	notReadyResponses int32
	// status code returned for meta_data.json once the instance is known
	metadataStatus int
	// whether the instance has userdata
	hasUserdata bool

	metadataRequests atomic.Int32
	userdataRequests atomic.Int32
}

func (s *metadataServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/" + metadataServiceMetadataPath:
		if s.metadataRequests.Add(1) <= s.notReadyResponses {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(s.metadataStatus)
		_, _ = w.Write([]byte(`{"uuid": "d8e02d56-2648-49a3-bf97-6be8f1204f38"}`))
	case "/" + userdataURLs[resource.IPv4].Path:
		s.userdataRequests.Add(1)
		ready := s.metadataRequests.Load() >= s.notReadyResponses
		if !ready || !s.hasUserdata {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(testUserdata))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *metadataServer) start(t *testing.T) url.URL {
	t.Helper()

	if s.metadataStatus == 0 {
		s.metadataStatus = http.StatusOK
	}
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	userdataURL := userdataURLs[resource.IPv4]
	userdataURL.Scheme = serverURL.Scheme
	userdataURL.Host = serverURL.Host
	return userdataURL
}

func newTestFetcher(t *testing.T) *resource.Fetcher {
	t.Helper()

	logger := log.New(true)
	fetcher := &resource.Fetcher{Logger: &logger}
	totalTimeout := 10
	if err := fetcher.UpdateHttpTimeoutsAndCAs(types.Timeouts{
		HTTPTotal: &totalTimeout,
	}, nil, types.Proxy{}); err != nil {
		t.Fatal(err)
	}
	return fetcher
}

func TestFetchUserdata(t *testing.T) {
	tests := []struct {
		name                 string
		server               metadataServer
		wantData             string
		wantErr              bool
		wantMetadataRequests int32
		wantUserdataRequests int32
	}{
		{
			name:                 "userdata",
			server:               metadataServer{hasUserdata: true},
			wantData:             testUserdata,
			wantUserdataRequests: 1,
		},
		{
			name:                 "no userdata",
			server:               metadataServer{},
			wantMetadataRequests: 1,
			wantUserdataRequests: 2,
		},
		{
			name:                 "userdata after metadata service becomes ready",
			server:               metadataServer{notReadyResponses: 2, hasUserdata: true},
			wantData:             testUserdata,
			wantMetadataRequests: 3,
			wantUserdataRequests: 2,
		},
		{
			name:                 "no userdata after metadata service becomes ready",
			server:               metadataServer{notReadyResponses: 2},
			wantMetadataRequests: 3,
			wantUserdataRequests: 2,
		},
		{
			name:                 "userdata without instance metadata",
			server:               metadataServer{metadataStatus: http.StatusNotFound, hasUserdata: true},
			wantData:             testUserdata,
			wantUserdataRequests: 1,
		},
		{
			name:                 "metadata request fails",
			server:               metadataServer{metadataStatus: http.StatusForbidden},
			wantErr:              true,
			wantMetadataRequests: 1,
			wantUserdataRequests: 1,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			userdataURL := tt.server.start(t)

			data, err := fetchUserdata(newTestFetcher(t), userdataURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v, want error %t", err, tt.wantErr)
			}
			if string(data) != tt.wantData {
				t.Errorf("got data %q, want %q", data, tt.wantData)
			}
			if got := tt.server.metadataRequests.Load(); got != tt.wantMetadataRequests {
				t.Errorf("got %d metadata requests, want %d", got, tt.wantMetadataRequests)
			}
			if got := tt.server.userdataRequests.Load(); got != tt.wantUserdataRequests {
				t.Errorf("got %d userdata requests, want %d", got, tt.wantUserdataRequests)
			}
		})
	}
}

func TestFetchConfigParallel(t *testing.T) {
	t.Run("waits for a slow endpoint after another one failed", func(t *testing.T) {
		failing := metadataServer{metadataStatus: http.StatusForbidden}
		slow := metadataServer{notReadyResponses: 2, hasUserdata: true}
		urls := []url.URL{failing.start(t), slow.start(t)}

		data, err := fetchConfigParallel(newTestFetcher(t), urls)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != testUserdata {
			t.Errorf("got data %q, want %q", data, testUserdata)
		}
	})

	t.Run("fails once every endpoint failed", func(t *testing.T) {
		first := metadataServer{metadataStatus: http.StatusForbidden}
		second := metadataServer{metadataStatus: http.StatusForbidden}
		urls := []url.URL{first.start(t), second.start(t)}

		data, err := fetchConfigParallel(newTestFetcher(t), urls)
		if err == nil {
			t.Fatalf("expected an error, got data %q", data)
		}
		for _, server := range []*metadataServer{&first, &second} {
			if got := server.metadataRequests.Load(); got != 1 {
				t.Errorf("got %d metadata requests, want 1", got)
			}
		}
	})
}
