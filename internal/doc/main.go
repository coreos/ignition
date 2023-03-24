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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos/go-semver/semver"

	v30 "github.com/coreos/ignition/v2/config/v3_0/types"
	v31 "github.com/coreos/ignition/v2/config/v3_1/types"
	v32 "github.com/coreos/ignition/v2/config/v3_2/types"
	v33 "github.com/coreos/ignition/v2/config/v3_3/types"
	v34 "github.com/coreos/ignition/v2/config/v3_4/types"
	v35 "github.com/coreos/ignition/v2/config/v3_5_experimental/types"
	doc "github.com/coreos/ignition/v2/internal/doc/generate"
)

func generate(dir string) error {
	configs := []struct {
		version string
		config  any
	}{
		{"3.0.0", v30.Config{}},
		{"3.1.0", v31.Config{}},
		{"3.2.0", v32.Config{}},
		{"3.3.0", v33.Config{}},
		{"3.4.0", v34.Config{}},
		{"3.5.0-experimental", v35.Config{}},
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	for _, c := range configs {
		ver := semver.New(c.version)

		// open file
		prerelease := ""
		if ver.PreRelease != "" {
			prerelease = "_" + string(ver.PreRelease)
		}
		path := filepath.Join(dir, fmt.Sprintf("configuration-v%d_%d%s.md", ver.Major, ver.Minor, prerelease))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		// write docs
		err = doc.Generate(c.config, f)
		if err != nil {
			return fmt.Errorf("generating doc for %s: %w", c.version, err)
		}
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		os.Exit(1)
	}
	if err := generate(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
