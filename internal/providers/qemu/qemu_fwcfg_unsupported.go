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

// +build s390x ppc64le

package qemu

import (
	"github.com/coreos/ignition/v2/internal/resource"
)

func fwCfgSupported() bool {
	return false
}

func fetchFromFwCfg(_ *resource.Fetcher) ([]byte, error) {
	panic("unreachable")
}
