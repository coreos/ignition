// Copyright 2021 Red Hat, Inc.
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

package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type State struct {
	// Information about configs fetched by the fetch stages.  Used
	// when writing the result file in files stage.
	FetchedConfigs []FetchedConfig `json:"fetchedConfigs"`
	// Key files generated during LUKS setup in disks stage, which need
	// to be written out during files stage.  files stage removes them
	// from state afterward to avoid leaking the keys into the running
	// system.
	LuksPersistKeyFiles map[string]string `json:"luksPersistKeyFiles"`
	// List of directories created by NotateMkdirAll(), relative to
	// the configured root dir.  Currently used to record directories
	// created by the mount stage so the files stage can chown them
	// when creating users.
	NotatedDirectories []string `json:"notatedDirectories"`
}

type FetchedConfig struct {
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	Referenced bool   `json:"referenced"`
}

func Load(path string) (State, error) {
	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		// valid; return empty struct
		return State{}, nil
	} else if err != nil {
		return State{}, fmt.Errorf("reading state file: %w", err)
	}
	var state State
	if err = json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parsing state file: %w", err)
	}
	return state, nil
}

func (s *State) Save(path string) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("serializing state file: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating directory for state file: %w", err)
	}
	if err := ioutil.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	return nil
}
