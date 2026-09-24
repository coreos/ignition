// Copyright 2026 Red Hat, Inc
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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func processFile(reader io.Reader, processor string) ([]byte, error) {
	if processor == "" {
		return io.ReadAll(reader)
	}

	cmd := exec.Command(processor)
	cmd.Stdin = reader
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("file processor %q failed: %w", processor, err)
	}
	return output, nil
}

func processLocalFile(path, processor string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	output, err := processFile(file, processor)
	return output, errors.Join(err, file.Close())
}
