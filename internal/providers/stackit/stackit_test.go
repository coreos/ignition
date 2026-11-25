// Copyright 2026 CoreOS, Inc.
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

package stackit

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/stretchr/testify/assert"
)

func TestFetchConfig(t *testing.T) {
	const validConfig = `{"ignition":{"version":"3.6.0"},"systemd":{"units":[{"name":"example.service","mask":true}]}}`
	tests := []struct {
		name         string
		status       int
		body         string
		retry        bool
		offline      bool
		wantErr      error
		wantFatal    bool
		wantRequests int32
	}{
		{
			name:         "valid config",
			status:       http.StatusOK,
			body:         validConfig,
			wantRequests: 1,
		},
		{
			name:         "missing userdata",
			status:       http.StatusNotFound,
			body:         "<html>Not Found</html>",
			wantErr:      errors.ErrEmpty,
			wantRequests: 1,
		},
		{
			name:         "empty userdata",
			status:       http.StatusOK,
			wantErr:      errors.ErrEmpty,
			wantRequests: 1,
		},
		{
			name:         "no content",
			status:       http.StatusNoContent,
			wantErr:      errors.ErrEmpty,
			wantRequests: 1,
		},
		{
			name:         "invalid config",
			status:       http.StatusOK,
			body:         `{"ignition":{"version":"3.6.0"},"storage":{"files":[{"path":"relative"}]}}`,
			wantErr:      errors.ErrInvalid,
			wantFatal:    true,
			wantRequests: 1,
		},
		{
			name:         "forbidden userdata",
			status:       http.StatusForbidden,
			body:         validConfig,
			wantErr:      resource.ErrFailed,
			wantRequests: 1,
		},
		{
			name:         "retry unavailable metadata service",
			status:       http.StatusOK,
			body:         validConfig,
			retry:        true,
			wantRequests: 2,
		},
		{
			name:    "offline",
			status:  http.StatusOK,
			body:    validConfig,
			offline: true,
			wantErr: resource.ErrNeedNet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := requests.Add(1)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/openstack/latest/user_data", r.URL.Path)
				if test.retry && attempt == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			originalURL := userdataURL
			t.Cleanup(func() { userdataURL = originalURL })
			userdataURL.Host = serverURL.Host

			logger := log.New(true)
			fetcher := resource.Fetcher{Logger: &logger, Offline: test.offline}
			timeout := 5
			if err := fetcher.UpdateHttpTimeoutsAndCAs(types.Timeouts{HTTPTotal: &timeout}, nil, types.Proxy{}); err != nil {
				t.Fatal(err)
			}

			cfg, rpt, err := fetchConfig(&fetcher)
			// The engine compares these errors directly, so preserve their identity.
			if err != test.wantErr {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			assert.Equal(t, test.wantFatal, rpt.IsFatal(), "unexpected report: %s", rpt.String())
			assert.Equal(t, test.wantRequests, requests.Load())
			if test.wantErr != nil {
				assert.Equal(t, types.Config{}, cfg)
				return
			}

			assert.Equal(t, types.MaxVersion.String(), cfg.Ignition.Version)
			if !assert.Len(t, cfg.Systemd.Units, 1) {
				return
			}
			assert.Equal(t, "example.service", cfg.Systemd.Units[0].Name)
			masked := true
			assert.Equal(t, &masked, cfg.Systemd.Units[0].Mask)
		})
	}
}
