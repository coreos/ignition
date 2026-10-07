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

package azure

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
	"golang.org/x/sys/unix"
)

func newIMDSTestFetcher(t *testing.T, handler http.Handler) *resource.Fetcher {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	// Tests using this helper must stay serial since the endpoint is package-global.
	originalURL := imdsUserdataURL
	imdsUserdataURL.Scheme = serverURL.Scheme
	imdsUserdataURL.Host = serverURL.Host
	t.Cleanup(func() { imdsUserdataURL = originalURL })

	logger := log.New(true)
	fetcher := &resource.Fetcher{Logger: &logger}
	totalTimeout := 5
	if err := fetcher.UpdateHttpTimeoutsAndCAs(types.Timeouts{
		HTTPTotal: &totalTimeout,
	}, nil, types.Proxy{}); err != nil {
		t.Fatal(err)
	}
	return fetcher
}

func TestFetchFromAzureMetadataEmptyUserData(t *testing.T) {
	ovfError := errors.New("OVF fetch failed")
	tests := []struct {
		name    string
		wantErr error
	}{
		{name: "returns OVF config"},
		{name: "returns OVF error", wantErr: ovfError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			fetcher := newIMDSTestFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusOK)
			}))

			wantConfig := types.Config{
				Ignition: types.Ignition{Version: "3.7.0-experimental"},
			}
			wantReport := report.Report{
				Entries: []report.Entry{{Kind: report.Warn, Message: "OVF report"}},
			}
			originalFetch := fetchFromOvfDevice
			t.Cleanup(func() { fetchFromOvfDevice = originalFetch })
			ovfCalls := 0
			fetchFromOvfDevice = func(f *resource.Fetcher, fsTypes []string) (types.Config, report.Report, error) {
				ovfCalls++
				if requests.Load() != 1 {
					t.Errorf("expected one IMDS request before OVF fallback, got %d", requests.Load())
				}
				if f != fetcher {
					t.Error("OVF fallback received a different fetcher")
				}
				if !reflect.DeepEqual(fsTypes, []string{CDS_FSTYPE_UDF}) {
					t.Errorf("expected UDF filesystem type, got %v", fsTypes)
				}
				return wantConfig, wantReport, tt.wantErr
			}

			gotConfig, gotReport, err := fetchFromAzureMetadata(fetcher)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if ovfCalls != 1 {
				t.Fatalf("expected one OVF fallback call, got %d", ovfCalls)
			}
			if requests.Load() != 1 {
				t.Fatalf("expected one IMDS request, got %d", requests.Load())
			}
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Fatalf("expected OVF config %+v, got %+v", wantConfig, gotConfig)
			}
			if !reflect.DeepEqual(gotReport, wantReport) {
				t.Fatalf("expected OVF report %+v, got %+v", wantReport, gotReport)
			}
		})
	}
}

// ovfEnvWithCustomData returns an Azure ovf-env.xml with the given CustomData
// element (which may be empty) spliced into the provisioning section.
func ovfEnvWithCustomData(customDataElement string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<ns0:Environment xmlns:ns0="http://schemas.dmtf.org/ovf/environment/1"
  xmlns:ns1="http://schemas.microsoft.com/windowsazure"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ns1:ProvisioningSection>
    <ns1:Version>1.0</ns1:Version>
    <ns1:LinuxProvisioningConfigurationSet>
      <ns1:ConfigurationSetType>LinuxProvisioningConfiguration</ns1:ConfigurationSetType>
      <ns1:HostName>host</ns1:HostName>
      <ns1:UserName>core</ns1:UserName>
      %s
      <ns1:DisableSshPasswordAuthentication>true</ns1:DisableSshPasswordAuthentication>
    </ns1:LinuxProvisioningConfigurationSet>
  </ns1:ProvisioningSection>
  <ns1:PlatformSettingsSection>
    <ns1:Version>1.0</ns1:Version>
    <ns1:PlatformSettings>
      <ns1:ProvisionGuestAgent>true</ns1:ProvisionGuestAgent>
    </ns1:PlatformSettings>
  </ns1:PlatformSettingsSection>
</ns0:Environment>`, customDataElement)
}

func TestCustomDataFromOvfEnv(t *testing.T) {
	config := `{"ignition":{"version":"3.4.0"}}`
	encoded := base64.StdEncoding.EncodeToString([]byte(config))

	tests := []struct {
		name    string
		xml     string
		out     string
		nilOut  bool
		wantErr error
	}{
		{
			name: "custom data present",
			xml:  ovfEnvWithCustomData(fmt.Sprintf("<ns1:CustomData>%s</ns1:CustomData>", encoded)),
			out:  config,
		},
		{
			name: "custom data split across lines",
			xml:  ovfEnvWithCustomData(fmt.Sprintf("<ns1:CustomData>%s\n      %s</ns1:CustomData>", encoded[:8], encoded[8:])),
			out:  config,
		},
		{
			name:   "no custom data element",
			xml:    ovfEnvWithCustomData(""),
			nilOut: true,
		},
		{
			name:   "empty custom data element",
			xml:    ovfEnvWithCustomData("<ns1:CustomData></ns1:CustomData>"),
			nilOut: true,
		},
		{
			name:    "malformed xml",
			xml:     "<ns0:Environment>not closed",
			wantErr: errParseOvfEnv,
		},
		{
			name:    "invalid base64",
			xml:     ovfEnvWithCustomData("<ns1:CustomData>!!! not base64 !!!</ns1:CustomData>"),
			wantErr: errDecodeCustomData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := customDataFromOvfEnv([]byte(tt.xml))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.nilOut {
				if got != nil {
					t.Fatalf("expected nil, got %q", got)
				}
				return
			}
			if string(got) != tt.out {
				t.Fatalf("expected %q, got %q", tt.out, got)
			}
		})
	}
}

func TestReadCustomData(test *testing.T) {
	config := `{"ignition":{"version":"3.4.0"}}`
	encoded := base64.StdEncoding.EncodeToString([]byte(config))
	ovf := ovfEnvWithCustomData(fmt.Sprintf("<ns1:CustomData>%s</ns1:CustomData>", encoded))
	binConfig := []byte(`{"ignition":{"version":"3.5.0"}}`)
	malformedOvf := "<ns0:Environment>not closed"

	tests := []struct {
		name    string
		xml     string
		bin     []byte
		binDir  bool
		out     string
		wantErr error
	}{
		{
			name: "ovf without bin",
			xml:  ovf,
			out:  config,
		},
		{
			name: "ovf preferred over bin",
			xml:  ovf,
			bin:  binConfig,
			out:  config,
		},
		{
			name: "missing custom data falls back to bin",
			xml:  ovfEnvWithCustomData(""),
			bin:  binConfig,
			out:  string(binConfig),
		},
		{
			name: "empty custom data falls back to bin",
			xml:  ovfEnvWithCustomData("<ns1:CustomData> \n\t </ns1:CustomData>"),
			bin:  binConfig,
			out:  string(binConfig),
		},
		{
			name: "malformed xml falls back to bin",
			xml:  malformedOvf,
			bin:  binConfig,
			out:  string(binConfig),
		},
		{
			name: "invalid base64 falls back to bin",
			xml:  ovfEnvWithCustomData("<ns1:CustomData>!!! not base64 !!!</ns1:CustomData>"),
			bin:  binConfig,
			out:  string(binConfig),
		},
		{
			name: "no custom data in either source",
			xml:  ovfEnvWithCustomData(""),
		},
		{
			name: "malformed ovf with missing bin is empty",
			xml:  malformedOvf,
		},
		{
			name: "malformed ovf with empty bin is empty",
			xml:  malformedOvf,
			bin:  []byte{},
		},
		{
			name:    "bin read error",
			xml:     ovfEnvWithCustomData(""),
			binDir:  true,
			wantErr: unix.EISDIR,
		},
		{
			name:    "bin alone is not a config drive",
			bin:     binConfig,
			wantErr: os.ErrNotExist,
		},
	}

	for _, testCase := range tests {
		test.Run(testCase.name, func(test *testing.T) {
			directory := test.TempDir()
			if testCase.xml != "" {
				if err := os.WriteFile(filepath.Join(directory, ovfEnvPath), []byte(testCase.xml), 0600); err != nil {
					test.Fatal(err)
				}
			}
			if testCase.binDir {
				if err := os.Mkdir(filepath.Join(directory, configPath), 0700); err != nil {
					test.Fatal(err)
				}
			} else if testCase.bin != nil {
				if err := os.WriteFile(filepath.Join(directory, configPath), testCase.bin, 0600); err != nil {
					test.Fatal(err)
				}
			}

			logger := log.New(true)
			got, err := readCustomData(&logger, directory)
			if !errors.Is(err, testCase.wantErr) {
				test.Fatalf("expected error %v, got %v", testCase.wantErr, err)
			}
			if string(got) != testCase.out {
				test.Fatalf("expected %q, got %q", testCase.out, got)
			}
		})
	}
}
