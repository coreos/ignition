// Copyright 2017 CoreOS, Inc.
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

package system

import (
	"io/ioutil"
	"os"
	"path/filepath"

	latest "github.com/coreos/ignition/v2/config/v3_3_experimental"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	userFilename = "user.ign"
)

// FetchBaseConfig fetches base config fragments from the `base.d` and platform config fragments from
// the `base.platform.d/platform`(if available), and merge them in the right order.
func FetchBaseConfig(logger *log.Logger, platformName string) (types.Config, report.Report, error) {
	fullBaseConfig, fullReport, err := fetchBaseDirectoryConfig(logger, "base.d")
	if err != nil {
		return types.Config{}, fullReport, err
	}

	platformDir := filepath.Join("base.platform.d", platformName)
	basePlatformDConfig, basePlatformDReport, err := fetchBaseDirectoryConfig(logger, platformDir)
	if err != nil {
		logger.Info("no config at %q: %v", platformDir, err)
	}
	fullBaseConfig = latest.Merge(fullBaseConfig, basePlatformDConfig)
	fullReport.Merge(basePlatformDReport)
	return fullBaseConfig, fullReport, nil
}

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	return fetchConfig(f.Logger, userFilename)
}

func fetchConfig(logger *log.Logger, filename string) (types.Config, report.Report, error) {
	path := filepath.Join(distro.SystemConfigDir(), filename)
	logger.Info("reading system config file %q", path)

	rawConfig, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		logger.Info("no config at %q", path)
		return types.Config{}, report.Report{}, providers.ErrNoProvider
	} else if err != nil {
		logger.Err("couldn't read config %q: %v", path, err)
		return types.Config{}, report.Report{}, err
	}
	return util.ParseConfig(logger, rawConfig)
}

// fetchBaseDirectoryConfig is a helper function to merge all the base config fragments inside of a particular directory.
func fetchBaseDirectoryConfig(logger *log.Logger, dir string) (types.Config, report.Report, error) {
	var baseConfig types.Config
	var report report.Report
	path := filepath.Join(distro.SystemConfigDir(), dir)
	configs, err := ioutil.ReadDir(path)
	if os.IsNotExist(err) {
		logger.Info("no config dir at %q", path)
		return types.Config{}, report, nil
	} else if err != nil {
		logger.Err("couldn't read config dir %q: %v", path, err)
		return types.Config{}, report, err
	}
	if len(configs) == 0 {
		logger.Info("no configs at %q", path)
		return types.Config{}, report, nil
	}
	for _, config := range configs {
		intermediateConfig, intermediateReport, err := fetchConfig(logger, filepath.Join(dir, config.Name()))
		if err != nil {
			return types.Config{}, intermediateReport, err
		}
		baseConfig = latest.Merge(baseConfig, intermediateConfig)
		report.Merge(intermediateReport)
	}
	return baseConfig, report, nil
}
