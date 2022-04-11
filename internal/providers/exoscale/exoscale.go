// Copyright 2020 CoreOS, Inc.
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

// The Exoscale provider fetches a remote configuration from the
// Exoscale user-data metadata service URL.

package exoscale

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/url"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	userdataURL = url.URL{
		Scheme: "http",
		Host:   "169.254.169.254",
		Path:   "1.0/user-data",
	}
)

// FetchConfig fetch Exoscale ign user-data config. The user-data
// can be delivered compressed (gzip) or uncompressed based on if
// the user used the exo CLI or the web UI. This means we must first
// get the data back and check to see if it's gzip compressed or not.
// https://github.com/coreos/fedora-coreos-tracker/issues/1160
func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := f.FetchToBuffer(userdataURL, resource.FetchOptions{})
	if err == resource.ErrNotFound {
		f.Logger.Info("Metadata service returned 404; no config")
		return types.Config{}, report.Report{}, errors.ErrEmpty
	} else if err != nil {
		return types.Config{}, report.Report{}, err
	}
	if len(data) == 0 {
		f.Logger.Info("Metadata service returned empty user-data; assuming no config")
		return types.Config{}, report.Report{}, errors.ErrEmpty
	}
	// Check for gzip file magic and decompress if found
	if len(data) > 2 && bytes.Equal(data[0:2], []byte{0x1f, 0x8b}) {
		f.Logger.Info("Detected gzip compression, attempting to decompress")
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return types.Config{}, report.Report{}, err
		}
		defer reader.Close()
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return types.Config{}, report.Report{}, err
		}
	}

	return util.ParseConfig(f.Logger, data)
}
