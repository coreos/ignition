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
