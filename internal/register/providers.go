// Copyright 2023 Red Hat, Inc.
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

package register

import (
	_ "github.com/coreos/ignition/v2/internal/providers/akamai"
	_ "github.com/coreos/ignition/v2/internal/providers/aliyun"
	_ "github.com/coreos/ignition/v2/internal/providers/applehv"
	_ "github.com/coreos/ignition/v2/internal/providers/aws"
	_ "github.com/coreos/ignition/v2/internal/providers/azure"
	_ "github.com/coreos/ignition/v2/internal/providers/azurestack"
	_ "github.com/coreos/ignition/v2/internal/providers/cloudstack"
	_ "github.com/coreos/ignition/v2/internal/providers/digitalocean"
	_ "github.com/coreos/ignition/v2/internal/providers/exoscale"
	_ "github.com/coreos/ignition/v2/internal/providers/file"
	_ "github.com/coreos/ignition/v2/internal/providers/gcp"
	_ "github.com/coreos/ignition/v2/internal/providers/hetzner"
	_ "github.com/coreos/ignition/v2/internal/providers/hyperv"
	_ "github.com/coreos/ignition/v2/internal/providers/ibmcloud"
	_ "github.com/coreos/ignition/v2/internal/providers/kubevirt"
	_ "github.com/coreos/ignition/v2/internal/providers/metal"
	_ "github.com/coreos/ignition/v2/internal/providers/nutanix"
	_ "github.com/coreos/ignition/v2/internal/providers/nvidiabluefield"
	_ "github.com/coreos/ignition/v2/internal/providers/openstack"
	_ "github.com/coreos/ignition/v2/internal/providers/oraclecloud"
	_ "github.com/coreos/ignition/v2/internal/providers/packet"
	_ "github.com/coreos/ignition/v2/internal/providers/powervs"
	_ "github.com/coreos/ignition/v2/internal/providers/proxmoxve"
	_ "github.com/coreos/ignition/v2/internal/providers/qemu"
	_ "github.com/coreos/ignition/v2/internal/providers/scaleway"
	_ "github.com/coreos/ignition/v2/internal/providers/upcloud"
	_ "github.com/coreos/ignition/v2/internal/providers/virtualbox"
	_ "github.com/coreos/ignition/v2/internal/providers/vmware"
	_ "github.com/coreos/ignition/v2/internal/providers/vultr"
	_ "github.com/coreos/ignition/v2/internal/providers/zvm"
)
