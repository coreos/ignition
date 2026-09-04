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

// Reads all markdown files in the specified directory and validates the
// Ignition and Butane configs wrapped in code fences.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	butane "github.com/coreos/ignition/v2/butane/config"
	butanecommon "github.com/coreos/ignition/v2/butane/config/common"
	ignition "github.com/coreos/ignition/v2/config"
)

// Specific section markers used in the docs to indicate that the Markdown code
// section right after one should be treated as being a valid config and thus
// used for testing.
const (
	ignitionMarker = "<!-- ignition -->"
	butaneMarker   = "<!-- butane-config -->"
)

// The kind of config a section holds, which determines both the code fence it
// must use and how it is validated.
type configKind int

const (
	ignitionConfig configKind = iota
	butaneConfig
)

func (k configKind) fence() string {
	if k == butaneConfig {
		return "```yaml"
	}
	return "```json"
}

func (k configKind) String() string {
	if k == butaneConfig {
		return "Butane"
	}
	return "Ignition"
}

// A config section extracted from a Markdown file.
type configSection struct {
	kind  configKind
	lines []string
}

// Represent the state we are in while trying to extract config sections from
// the examples in the docs.
type sectionState int

const (
	notInSection sectionState = iota
	expectingSection
	inSection
)

func main() {
	flags := struct {
		help     bool
		root     string
		filesDir string
	}{}

	flag.BoolVar(&flags.help, "help", false, "Print help and exit.")
	flag.StringVar(&flags.root, "root", "docs", "Path to the documentation.")
	flag.StringVar(&flags.filesDir, "files-dir", "", "Directory Butane configs may embed local files from.")

	flag.Parse()

	if flags.help {
		flag.Usage()
		return
	}

	if err := filepath.Walk(flags.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".md") || info.IsDir() {
			return nil
		}

		fileContents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fileLines := strings.Split(string(fileContents), "\n")
		sections, ignored, err := findConfigSections(fileLines)
		if err != nil {
			return fmt.Errorf("invalid section formatting in %s: %s", path, err)
		}
		if len(sections) != 0 {
			fmt.Printf("Found %d sections in: %s\n", len(sections), path)
		}
		if ignored != 0 {
			fmt.Printf("Ignored %d partial or empty sections in: %s\n", ignored, path)
		}

		for _, section := range sections {
			if err := validateSection(section, info.Name(), flags.filesDir); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed while validating docs: %v\n", err)
		os.Exit(1)
	}
}

// validateSection checks one config section, matching the strictness of
// `ignition-validate` and of `butane --check --strict` respectively.
func validateSection(section configSection, name, filesDir string) error {
	cfg := strings.Join(section.lines, "\n")

	switch section.kind {
	case butaneConfig:
		_, r, err := butane.TranslateBytes([]byte(cfg), butanecommon.TranslateBytesOptions{
			TranslateOptions: butanecommon.TranslateOptions{
				FilesDir: filesDir,
			},
		})
		if err != nil {
			return fmt.Errorf("fatal error translating %s: %s\nConfig:\n%s", name, err, cfg)
		}
		// `--strict` treats any report entry, warnings included, as fatal
		if len(r.Entries) > 0 {
			return fmt.Errorf("non-empty translation report in %s: %s\nConfig:\n%s", name, r.String(), cfg)
		}
	default:
		_, r, err := ignition.Parse([]byte(cfg))
		// the report provides a more specific error
		// description, so check that first
		reportStr := r.String()
		if reportStr != "" {
			return fmt.Errorf("non-empty parsing report in %s: %s\nConfig:\n%s", name, reportStr, cfg)
		}
		if err != nil {
			return fmt.Errorf("fatal error parsing %s: %s\nConfig:\n%s", name, err, cfg)
		}
	}

	return nil
}

func findConfigSections(fileLines []string) ([]configSection, uint, error) {
	var sections []configSection
	var currentSection []string
	var currentKind configKind

	var ignoredSections uint = 0
	var state = notInSection

	for _, line := range fileLines {
		switch state {
		case notInSection:
			switch line {
			case ignitionMarker:
				currentKind = ignitionConfig
				state = expectingSection
			case butaneMarker:
				currentKind = butaneConfig
				state = expectingSection
			}

		case expectingSection:
			if line == currentKind.fence() {
				state = inSection
			} else {
				return sections, ignoredSections, fmt.Errorf("expecting '%s', found: %s", currentKind.fence(), line)
			}

		case inSection:
			if line == "```" {
				if len(currentSection) == 0 || currentSection[0] == "..." {
					// Ignore empty sections and sections that are not full configs
					ignoredSections++
				} else {
					sections = append(sections, configSection{
						kind:  currentKind,
						lines: currentSection,
					})
				}
				currentSection = nil
				state = notInSection
			} else {
				currentSection = append(currentSection, line)
			}
		}
	}

	// A file ending mid-section would otherwise drop that section and report
	// success, silently skipping validation of it.
	switch state {
	case expectingSection:
		return sections, ignoredSections, fmt.Errorf("expecting '%s' after %s marker, found end of file", currentKind.fence(), currentKind)
	case inSection:
		return sections, ignoredSections, fmt.Errorf("unterminated %s config section", currentKind)
	}

	return sections, ignoredSections, nil
}
