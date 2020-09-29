// Copyright 2020 Red Hat, Inc.
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

// The AzureStack provider is a wrapper around the regular Azure provider
// but allows for the use of both UDF and iso9660 filesystems on the OVF disk.

package azurestack

import (
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers/azure"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/vcontext/report"
)

// These constants are the types of CDROM filesystems that might
// be used to present a custom-data volume. Azure proper uses a
// udf volume, while Azure Stack might use udf or iso9660.
const (
	CDS_FSTYPE_UDF     = "udf"
	CDS_FSTYPE_ISO9660 = "iso9660"
)

// FetchConfig implements the platform.NewFetcher interface.
func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	return azure.FetchFromOvfDevice(f, []string{CDS_FSTYPE_UDF, CDS_FSTYPE_ISO9660})
}
