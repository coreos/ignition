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

// The gcp provider fetches a remote configuration from the GCE user-data
// metadata service URL.

package gcp

import (
	"net/http"
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	userdataUrl = url.URL{
		Scheme: "http",
		Host:   "metadata.google.internal",
		Path:   "computeMetadata/v1/instance/attributes/user-data",
	}
	metadataHeaderKey = "Metadata-Flavor"
	metadataHeaderVal = "Google"
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	headers := make(http.Header)
	headers.Set(metadataHeaderKey, metadataHeaderVal)
	data, err := f.FetchToBuffer(userdataUrl, resource.FetchOptions{
		Headers: headers,
	})
	if err != nil && err != resource.ErrNotFound {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}
