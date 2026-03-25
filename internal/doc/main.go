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
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/coreos/go-semver/semver"

	"github.com/coreos/ignition/v2/config/doc"
	v30 "github.com/coreos/ignition/v2/config/v3_0/types"
	v31 "github.com/coreos/ignition/v2/config/v3_1/types"
	v32 "github.com/coreos/ignition/v2/config/v3_2/types"
	v33 "github.com/coreos/ignition/v2/config/v3_3/types"
	v34 "github.com/coreos/ignition/v2/config/v3_4/types"
	v35 "github.com/coreos/ignition/v2/config/v3_5/types"
	v36 "github.com/coreos/ignition/v2/config/v3_6/types"
	v37 "github.com/coreos/ignition/v2/config/v3_7_experimental/types"
)

var (
	//go:embed header.md
	headerRaw string
	header    = template.Must(template.New("header").Parse(headerRaw))
)

func generate(dir string) error {
	configs := []struct {
		version string
		config  any
	}{
		// generate in inverse order of website navbar
		{"3.7.0-experimental", v37.Config{}},
		{"3.0.0", v30.Config{}},
		{"3.1.0", v31.Config{}},
		{"3.2.0", v32.Config{}},
		{"3.3.0", v33.Config{}},
		{"3.4.0", v34.Config{}},
		{"3.5.0", v35.Config{}},
		{"3.6.0", v36.Config{}},
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// parse input config
	comps, err := doc.IgnitionComponents()
	if err != nil {
		return err
	}

	for i, c := range configs {
		if err := func() (err error) {
			ver := *semver.New(c.version)

			// clean up any previous experimental spec doc, for use
			// during spec stabilization
			experimentalPath := filepath.Join(dir, fmt.Sprintf("configuration-v%d_%d_experimental.md", ver.Major, ver.Minor))
			if err := os.Remove(experimentalPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}

			// open file
			var path string
			switch ver.PreRelease {
			case "":
				path = filepath.Join(dir, fmt.Sprintf("configuration-v%d_%d.md", ver.Major, ver.Minor))
			case "experimental":
				path = experimentalPath
			default:
				panic(fmt.Errorf("unexpected prerelease: %v", ver.PreRelease))
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, f.Close())
			}()

			// write header
			args := struct {
				Version  semver.Version
				NavOrder int
			}{
				Version:  ver,
				NavOrder: 50 - i,
			}
			if err := header.Execute(f, args); err != nil {
				return fmt.Errorf("writing header for %s: %w", c.version, err)
			}

			// write docs
			vers := doc.VariantVersions{
				doc.IGNITION_VARIANT: ver,
			}
			if err := comps.Generate(vers, c.config, nil, f); err != nil {
				return fmt.Errorf("generating doc for %s: %w", c.version, err)
			}

			return nil
		}(); err != nil {
			return err
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
