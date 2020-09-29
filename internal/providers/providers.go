// Copyright 2015 CoreOS, Inc.
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

package providers

import (
	"errors"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	ErrNoProvider = errors.New("config provider was not online")
)

type FuncFetchConfig func(f *resource.Fetcher) (types.Config, report.Report, error)
type FuncInit func(f *resource.Fetcher) error
type FuncNewFetcher func(logger *log.Logger) (resource.Fetcher, error)
type FuncPostStatus func(stageName string, f resource.Fetcher, e error) error
