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
	"os"
	"path/filepath"
	"sort"

	latest "github.com/coreos/ignition/v2/config/v3_7_experimental"
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

const (
	userFilename = "user.ign"
)

var (
	// we are a special-cased system provider; don't register ourselves
	// for lookup by name
	Config = platform.NewConfig(platform.Provider{
		Name:  "system",
		Fetch: fetchConfig,
	})
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

// fetchConfig searches for user.ign across system config directories
// in priority order; first found wins.
func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	for _, dir := range distro.SystemConfigDirs() {
		cfg, r, err := readConfigFile(f.Logger, filepath.Join(dir, userFilename))
		if err == platform.ErrNoProvider {
			continue
		}
		return cfg, r, err
	}
	return types.Config{}, report.Report{}, platform.ErrNoProvider
}

// readConfigFile reads and parses a config at the given path.
// Returns ErrNoProvider if the file does not exist.
func readConfigFile(logger *log.Logger, path string) (types.Config, report.Report, error) {
	logger.Info("reading system config file %q", path)

	rawConfig, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		logger.Info("no config at %q", path)
		return types.Config{}, report.Report{}, platform.ErrNoProvider
	} else if err != nil {
		logger.Err("couldn't read config %q: %v", path, err)
		return types.Config{}, report.Report{}, err
	}
	return util.ParseConfig(logger, rawConfig)
}

// fetchBaseDirectoryConfig collects base config fragments from a subdirectory
// across all system config dirs. SystemConfigDirs() returns directories in
// descending priority order by construction (runtime > local > vendor), so
// iterating forward visits the highest-priority directory first; the first
// directory to claim a filename wins.
func fetchBaseDirectoryConfig(logger *log.Logger, dir string) (types.Config, report.Report, error) {
	fileMap := make(map[string]string)
	for _, sysDir := range distro.SystemConfigDirs() {
		path := filepath.Join(sysDir, dir)
		entries, err := os.ReadDir(path)
		if os.IsNotExist(err) {
			logger.Info("no config dir at %q", path)
			continue
		} else if err != nil {
			logger.Err("couldn't read config dir %q: %v", path, err)
			return types.Config{}, report.Report{}, err
		}
		for _, entry := range entries {
			if _, exists := fileMap[entry.Name()]; !exists {
				fileMap[entry.Name()] = filepath.Join(path, entry.Name())
			}
		}
	}

	if len(fileMap) == 0 {
		logger.Info("no configs found in %q across system config directories", dir)
		return types.Config{}, report.Report{}, nil
	}

	names := make([]string, 0, len(fileMap))
	for name := range fileMap {
		names = append(names, name)
	}
	sort.Strings(names)

	var baseConfig types.Config
	var fullReport report.Report
	for _, name := range names {
		intermediateConfig, intermediateReport, err := readConfigFile(logger, fileMap[name])
		if err != nil {
			return types.Config{}, intermediateReport, err
		}
		baseConfig = latest.Merge(baseConfig, intermediateConfig)
		fullReport.Merge(intermediateReport)
	}
	return baseConfig, fullReport, nil
}
