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
	"testing"
)

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
