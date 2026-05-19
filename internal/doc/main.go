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
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/ignition/v2/config/doc"
	"github.com/coreos/ignition/v2/config/util"
	"gopkg.in/yaml.v3"

	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	buUtil "github.com/coreos/butane/config/util"

	base0_3 "github.com/coreos/butane/base/v0_3"
	fcos1_0 "github.com/coreos/butane/config/fcos/v1_0"
	fcos1_1 "github.com/coreos/butane/config/fcos/v1_1"
	fcos1_2 "github.com/coreos/butane/config/fcos/v1_2"
	fcos1_3 "github.com/coreos/butane/config/fcos/v1_3"
	fcos1_4 "github.com/coreos/butane/config/fcos/v1_4"
	fcos1_5 "github.com/coreos/butane/config/fcos/v1_5"
	fcos1_6 "github.com/coreos/butane/config/fcos/v1_6"
	fcos1_7 "github.com/coreos/butane/config/fcos/v1_7"
	fcos1_8_exp "github.com/coreos/butane/config/fcos/v1_8_exp"
	fiot1_0 "github.com/coreos/butane/config/fiot/v1_0"
	fiot1_1_exp "github.com/coreos/butane/config/fiot/v1_1_exp"
	flatcar1_0 "github.com/coreos/butane/config/flatcar/v1_0"
	flatcar1_1 "github.com/coreos/butane/config/flatcar/v1_1"
	flatcar1_2_exp "github.com/coreos/butane/config/flatcar/v1_2_exp"
	openshift4_10 "github.com/coreos/butane/config/openshift/v4_10"
	openshift4_11 "github.com/coreos/butane/config/openshift/v4_11"
	openshift4_12 "github.com/coreos/butane/config/openshift/v4_12"
	openshift4_13 "github.com/coreos/butane/config/openshift/v4_13"
	openshift4_14 "github.com/coreos/butane/config/openshift/v4_14"
	openshift4_15 "github.com/coreos/butane/config/openshift/v4_15"
	openshift4_16 "github.com/coreos/butane/config/openshift/v4_16"
	openshift4_17 "github.com/coreos/butane/config/openshift/v4_17"
	openshift4_18 "github.com/coreos/butane/config/openshift/v4_18"
	openshift4_19 "github.com/coreos/butane/config/openshift/v4_19"
	openshift4_20 "github.com/coreos/butane/config/openshift/v4_20"
	openshift4_21 "github.com/coreos/butane/config/openshift/v4_21"
	openshift4_22 "github.com/coreos/butane/config/openshift/v4_22"
	openshift4_23_exp "github.com/coreos/butane/config/openshift/v4_23_exp"
	openshift4_8 "github.com/coreos/butane/config/openshift/v4_8"
	openshift4_9 "github.com/coreos/butane/config/openshift/v4_9"
	r4e1_0 "github.com/coreos/butane/config/r4e/v1_0"
	r4e1_1 "github.com/coreos/butane/config/r4e/v1_1"
	r4e1_2_exp "github.com/coreos/butane/config/r4e/v1_2_exp"
)

var (
	//go:embed header.md
	headerRaw string
	header    = template.Must(template.New("header").Parse(headerRaw))

	//go:embed butane.yaml
	butaneDocs []byte
)

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

type variant struct {
	desc     string
	variant  string
	versions []version
}

type version struct {
	version string
	config  buUtil.Config
}

func generate(dir string) error {
	configs := []variant{
		// alphabetical order
		{
			"Fedora CoreOS",
			"fcos",
			[]version{
				// inverse order of website navbar
				{"1.8.0-experimental", fcos1_8_exp.Config{}},
				{"1.0.0", fcos1_0.Config{}},
				{"1.1.0", fcos1_1.Config{}},
				{"1.2.0", fcos1_2.Config{}},
				{"1.3.0", fcos1_3.Config{}},
				{"1.4.0", fcos1_4.Config{}},
				{"1.5.0", fcos1_5.Config{}},
				{"1.6.0", fcos1_6.Config{}},
				{"1.7.0", fcos1_7.Config{}},
			},
		},
		{
			"Flatcar",
			"flatcar",
			[]version{
				// inverse order of website navbar
				{"1.2.0-experimental", flatcar1_2_exp.Config{}},
				{"1.0.0", flatcar1_0.Config{}},
				{"1.1.0", flatcar1_1.Config{}},
			},
		},
		{
			"OpenShift",
			"openshift",
			[]version{
				// inverse order of website navbar
				{"4.23.0-experimental", openshift4_23_exp.Config{}},
				{"4.8.0", openshift4_8.Config{}},
				{"4.9.0", openshift4_9.Config{}},
				{"4.10.0", openshift4_10.Config{}},
				{"4.11.0", openshift4_11.Config{}},
				{"4.12.0", openshift4_12.Config{}},
				{"4.13.0", openshift4_13.Config{}},
				{"4.14.0", openshift4_14.Config{}},
				{"4.15.0", openshift4_15.Config{}},
				{"4.16.0", openshift4_16.Config{}},
				{"4.17.0", openshift4_17.Config{}},
				{"4.18.0", openshift4_18.Config{}},
				{"4.19.0", openshift4_19.Config{}},
				{"4.20.0", openshift4_20.Config{}},
				{"4.21.0", openshift4_21.Config{}},
				{"4.22.0", openshift4_22.Config{}},
			},
		},
		{
			"RHEL for Edge",
			"r4e",
			[]version{
				// inverse order of website navbar
				{"1.2.0-experimental", r4e1_2_exp.Config{}},
				{"1.0.0", r4e1_0.Config{}},
				{"1.1.0", r4e1_1.Config{}},
			},
		},
		{
			"Fedora IoT",
			"fiot",
			[]version{
				// inverse order of website navbar
				{"1.1.0-experimental", fiot1_1_exp.Config{}},
				{"1.0.0", fiot1_0.Config{}},
			},
		},
	}

	// parse and snakify Ignition components
	comps, err := doc.IgnitionComponents()
	if err != nil {
		return err
	}
	for name, comp := range comps {
		snakify(&comp)
		comps[name] = comp
	}

	// parse and merge Butane DocFile
	butaneComps, err := doc.ParseComponents(bytes.NewBuffer(butaneDocs))
	if err != nil {
		return err
	}
	if err := comps.Merge(butaneComps); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	for i, variant := range configs {
		for j, version := range variant.versions {
			if err := generateOne(dir, comps, variant, version, 50*(i+1)-j); err != nil {
				return fmt.Errorf("generating docs for %s %s: %w", variant.variant, version.version, err)
			}
		}
	}
	return nil
}

func snakify(node *doc.DocNode) {
	node.Name = buUtil.Snake(node.Name)
	for i := range node.Children {
		snakify(&node.Children[i])
	}
}

func generateOne(dir string, comps doc.Components, variant variant, version version, navOrder int) error {
	ver := *semver.New(version.version)

	// clean up any previous experimental spec doc, for
	// use during spec stabilization
	experimentalPath := filepath.Join(dir, fmt.Sprintf("config-%s-v%d_%d-exp.md", variant.variant, ver.Major, ver.Minor))
	if err := os.Remove(experimentalPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	// open file
	var path string
	switch ver.PreRelease {
	case "":
		path = filepath.Join(dir, fmt.Sprintf("config-%s-v%d_%d.md", variant.variant, ver.Major, ver.Minor))
	case "experimental":
		path = experimentalPath
	default:
		panic(fmt.Errorf("unexpected prerelease: %v", ver.PreRelease))
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// write header
	args := struct {
		Variant  string
		Version  semver.Version
		NavOrder int
	}{
		Variant:  variant.desc,
		Version:  ver,
		NavOrder: navOrder,
	}
	if err := header.Execute(f, args); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	// write docs
	ignVer, err := getIgnitionVersion(variant.variant, ver)
	if err != nil {
		return err
	}
	vers := doc.VariantVersions{
		doc.IGNITION_VARIANT: ignVer,
		variant.variant:      ver,
	}
	ignore := func(path []string) bool {
		filters := version.config.FieldFilters()
		if filters == nil {
			return false
		}
		var camelPath []string
		for _, el := range path {
			camelPath = append(camelPath, buUtil.Camel(el))
		}
		pathStr := strings.Join(camelPath, ".")
		if variant.variant == "openshift" {
			pathStr = fmt.Sprintf("spec.config.%s", pathStr)
		}
		return filters.Lookup(pathStr) != nil
	}
	if err := comps.Generate(vers, version.config, ignore, f); err != nil {
		return fmt.Errorf("generating: %w", err)
	}
	return nil
}

func getIgnitionVersion(variant string, version semver.Version) (semver.Version, error) {
	// generate an empty Butane config with this variant/version
	// use a random OpenShift spec as a representative structure
	bu, err := yaml.Marshal(openshift4_13.Config{
		Config: fcos1_3.Config{
			Config: base0_3.Config{
				Variant: variant,
				Version: version.String(),
			},
		},
		Metadata: openshift4_13.Metadata{
			Name: "name",
			Labels: map[string]string{
				openshift4_13.ROLE_LABEL_KEY: "value",
			},
		},
	})
	if err != nil {
		return semver.Version{}, fmt.Errorf("generating skeleton Butane config: %w", err)
	}

	// translate to Ignition config
	ign, _, err := config.TranslateBytes(bu, common.TranslateBytesOptions{
		Raw: true,
	})
	if err != nil {
		return semver.Version{}, fmt.Errorf("translating skeleton Butane config: %w", err)
	}

	// parse Ignition version
	ver, _, err := util.GetConfigVersion(ign)
	if err != nil {
		return semver.Version{}, fmt.Errorf("getting Ignition config version: %w", err)
	}

	return ver, nil
}
