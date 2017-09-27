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
// Container Linux Configs wrapped in code fences.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/container-linux-config-transpiler/config"
)

const (
	infoString = "yaml container-linux-config"
)

func main() {
	flags := struct {
		help bool
		root string
	}{}

	flag.BoolVar(&flags.help, "help", false, "Print help and exit.")
	flag.StringVar(&flags.root, "root", "doc", "Path to the documentation.")

	flag.Parse()

	if flags.help {
		flag.Usage()
		return
	}

	if err := filepath.Walk(flags.root, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(info.Name(), ".md") || info.IsDir() {
			return nil
		}

		fileContents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		fileLines := strings.Split(string(fileContents), "\n")
		yamlSections := findYamlSections(fileLines)

		for _, yaml := range yamlSections {
			_, _, report := config.Parse([]byte(strings.Join(yaml, "\n")))
			reportStr := report.String()
			if reportStr != "" {
				return fmt.Errorf("non-empty parsing report in %s: %s", info.Name(), reportStr)
			}
		}

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed while validating docs: %v", err)
		os.Exit(1)
	}
}

func findYamlSections(fileLines []string) [][]string {
	var yamlSections [][]string
	var currentSection []string
	inASection := false
	for _, line := range fileLines {
		if line == "```" {
			inASection = false
			if len(currentSection) > 0 {
				yamlSections = append(yamlSections, currentSection)
			}
			currentSection = nil
		}
		if inASection {
			currentSection = append(currentSection, line)
		}
		if line == "```"+infoString {
			inASection = true
		}
	}
	return yamlSections
}
