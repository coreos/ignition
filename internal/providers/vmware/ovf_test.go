// Copyright 2014-2015 VMware, Inc. All Rights Reserved.
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

// Originally from https://github.com/vmware-archive/vmw-ovflib

package vmware

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var data_vsphere = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Environment
     xmlns="http://schemas.dmtf.org/ovf/environment/1"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xmlns:oe="http://schemas.dmtf.org/ovf/environment/1"
     xmlns:ve="http://www.vmware.com/schema/ovfenv"
     oe:id=""
     ve:vCenterId="vm-12345">
   <PlatformSection>
      <Kind>VMware ESXi</Kind>
      <Version>5.5.0</Version>
      <Vendor>VMware, Inc.</Vendor>
      <Locale>en</Locale>
   </PlatformSection>
   <PropertySection>
         <Property oe:key="foo" oe:value="42"/>
         <Property oe:key="bar" oe:value="0"/>
   </PropertySection>
   <ve:EthernetAdapterSection>
      <ve:Adapter ve:mac="00:00:00:00:00:00" ve:network="foo" ve:unitNumber="7"/>
   </ve:EthernetAdapterSection>
</Environment>`)

var data_vapprun = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Environment xmlns="http://schemas.dmtf.org/ovf/environment/1"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xmlns:oe="http://schemas.dmtf.org/ovf/environment/1"
     oe:id="CoreOS-vmw">
   <PlatformSection>
      <Kind>vapprun</Kind>
      <Version>1.0</Version>
      <Vendor>VMware, Inc.</Vendor>
      <Locale>en_US</Locale>
   </PlatformSection>
   <PropertySection>
      <Property oe:key="foo" oe:value="42"/>
      <Property oe:key="bar" oe:value="0"/>
      <Property oe:key="guestinfo.user_data.url" oe:value="https://gist.githubusercontent.com/sigma/5a64aac1693da9ca70d2/raw/plop.yaml"/>
      <Property oe:key="guestinfo.user_data.doc" oe:value=""/>
      <Property oe:key="guestinfo.meta_data.url" oe:value=""/>
      <Property oe:key="guestinfo.meta_data.doc" oe:value=""/>
   </PropertySection>
</Environment>`)

var data_delete_prop_simple = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Environment xmlns="http://schemas.dmtf.org/ovf/environment/1" xmlns:a="http://schemas.dmtf.org/ovf/environment/1">
   <Invalid>
     <InvalidKey>garbage!</InvalidKey>
   </Invalid>
   <PropertySection>
      <!-- XML attributes don't default to the default namespace -->
      <Property a:key="guestinfo.ignition.config.data" value="remove"/>
      <Property a:key="guestinfo.ignition.config.data.encoding" value="remove"/>
   </PropertySection>
</Environment>`)

var data_delete_prop_complex = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Environment xmlns="http://schemas.dmtf.org/ovf/environment/1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:oe="http://schemas.dmtf.org/ovf/environment/1" xmlns:ex="http://example.com/xmlns" oe:id="ignition-vmw">
   <PlatformSection>
      <Kind>vapprun</Kind>
      <Version>1.0</Version>
      <Vendor>VMware, Inc.</Vendor>
      <Locale>en_US</Locale>
   </PlatformSection>
   <Invalid>
     <InvalidKey>garbage!</InvalidKey>
   </Invalid>
   <PropertySection>
      <Property ex:key="guestinfo.ignition.config.data.encoding" oe:value="keep"/>
      <ex:Property oe:key="guestinfo.ignition.config.data.encoding" oe:value="keep"/>
      <Property oe:key="guestinfo.ignition.config.data" oe:value="remove"/>
      <Property oe:key="guestinfo.ignition.config.data.encoding" oe:value="remove"/>
      <Property oe:key="bar" oe:value="0"/>
   </PropertySection>
   <ex:PropertySection>
      <Property oe:key="guestinfo.ignition.config.data" oe:value="keep"/>
      <Property oe:key="guestinfo.ignition.config.data.encoding" oe:value="keep"/>
   </ex:PropertySection>
</Environment>`)

func TestOvfEnvProperties(t *testing.T) {
	var testOne = func(env_str []byte) {
		env, err := ReadOvfEnvironment(env_str)
		assert.Nil(t, err)
		props := env.Properties

		var val string
		var ok bool
		val, ok = props["foo"]
		assert.True(t, ok)
		assert.Equal(t, val, "42")

		val, ok = props["bar"]
		assert.True(t, ok)
		assert.Equal(t, val, "0")
	}

	testOne(data_vapprun)
	testOne(data_vsphere)
}

func TestOvfEnvPlatform(t *testing.T) {
	env, err := ReadOvfEnvironment(data_vsphere)
	assert.Nil(t, err)
	platform := env.Platform

	assert.Equal(t, platform.Kind, "VMware ESXi")
	assert.Equal(t, platform.Version, "5.5.0")
	assert.Equal(t, platform.Vendor, "VMware, Inc.")
	assert.Equal(t, platform.Locale, "en")
}

func TestVappRunUserDataUrl(t *testing.T) {
	env, err := ReadOvfEnvironment(data_vapprun)
	assert.Nil(t, err)
	props := env.Properties

	var val string
	var ok bool

	val, ok = props["guestinfo.user_data.url"]
	assert.True(t, ok)
	assert.Equal(t, val, "https://gist.githubusercontent.com/sigma/5a64aac1693da9ca70d2/raw/plop.yaml")
}

func TestInvalidData(t *testing.T) {
	_, err := ReadOvfEnvironment(append(data_vsphere, []byte("garbage")...))
	assert.Nil(t, err)
}

func TestDeleteProps(t *testing.T) {
	for _, sample := range [][]byte{data_delete_prop_simple, data_delete_prop_complex, data_vapprun} {
		var expectedLines []string
		expectedDelete := false
		for _, line := range strings.Split(string(sample), "\n") {
			if strings.Contains(line, "remove") {
				// drop XML element, leave indentation
				startSkip := strings.IndexAny(line, "<>")
				endSkip := strings.LastIndexAny(line, "<>")
				line = line[:startSkip] + line[endSkip+1:]
				expectedDelete = true
			}
			expectedLines = append(expectedLines, line)
		}
		expected := strings.Join(expectedLines, "\n")

		result, deleted, err := DeleteOvfProperties(sample, []string{"guestinfo.ignition.config.data", "guestinfo.ignition.config.data.encoding"})
		assert.Nil(t, err)
		assert.Equal(t, expected, string(result))
		assert.Equal(t, expectedDelete, deleted)
	}
}
