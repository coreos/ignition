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

package util

import (
	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"

	"github.com/vincent-petithory/dataurl"
)

// MakeProviderOutputFile is a helper function to create an output File for
// FetchWithFiles.
func MakeProviderOutputFile(path string, mode int, data []byte) types.File {
	url := dataurl.EncodeBytes(data)
	return types.File{
		Node: types.Node{
			Path: path,
			// Ignition is not designed to run twice, but don't
			// introduce a hard failure if it does
			Overwrite: util.BoolToPtr(true),
		},
		FileEmbedded1: types.FileEmbedded1{
			Contents: types.Resource{
				Source: &url,
			},
			Mode: &mode,
		},
	}
}
