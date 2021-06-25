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
// Ignition configs wrapped in code fences.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/ignition/v2/config"
)

// Specific section marker used in the docs to indicate that the Markdown code
// section right after it should be treated as being a valid Ignition config
// and thus used for testing.
const (
	sectionMarker = "<!-- ignition -->"
)

// Represent the state we are in while trying to extract Ignition config
// sections from the examples in the docs.
type sectionState int

const (
	notInSection sectionState = iota
	expectingSection
	inSection
)

func main() {
	flags := struct {
		help bool
		root string
	}{}

	flag.BoolVar(&flags.help, "help", false, "Print help and exit.")
	flag.StringVar(&flags.root, "root", "docs", "Path to the documentation.")

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

		fileContents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		fileLines := strings.Split(string(fileContents), "\n")
		jsonSections, ignored, err := findJsonSections(fileLines)
		if err != nil {
			return fmt.Errorf("invalid section formatting in %s: %s", path, err)
		}
		if len(jsonSections) != 0 {
			fmt.Printf("Found %d sections in: %s\n", len(jsonSections), path)
		}
		if ignored != 0 {
			fmt.Printf("Ignored %d partial or empty sections in: %s\n", ignored, path)
		}

		for _, json := range jsonSections {
			cfg := strings.Join(json, "\n")
			_, r, err := config.Parse([]byte(cfg))
			// the report provides a more specific error
			// description, so check that first
			reportStr := r.String()
			if reportStr != "" {
				return fmt.Errorf("non-empty parsing report in %s: %s\nConfig:\n%s", info.Name(), reportStr, cfg)
			}
			if err != nil {
				return fmt.Errorf("fatal error parsing %s: %s\nConfig:\n%s", info.Name(), err, cfg)
			}
		}

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed while validating docs: %v\n", err)
		os.Exit(1)
	}
}

func findJsonSections(fileLines []string) ([][]string, uint, error) {
	var jsonSections [][]string
	var currentSection []string

	var ignoredSections uint = 0
	var state sectionState = notInSection

	for _, line := range fileLines {
		switch state {
		case notInSection:
			if line == sectionMarker {
				state = expectingSection
			}

		case expectingSection:
			if line == "```json" {
				state = inSection
			} else {
				return jsonSections, ignoredSections, fmt.Errorf("expecting '```json', found: %s", line)
			}

		case inSection:
			if line == "```" {
				if len(currentSection) == 0 || currentSection[0] == "..." {
					// Ignore empty sections and sections that are not full configs
					ignoredSections++
				} else {
					jsonSections = append(jsonSections, currentSection)
				}
				currentSection = nil
				state = notInSection
			} else {
				currentSection = append(currentSection, line)
			}
		}
	}
	return jsonSections, ignoredSections, nil
}
