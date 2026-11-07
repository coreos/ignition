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
	"path/filepath"
	"strings"
)

func parseFileProcessorOptions(processor, extensions string, processorSet, extensionsSet bool) ([]string, error) {
	if !processorSet && !extensionsSet {
		return nil, nil
	}
	if !processorSet {
		return nil, errors.New("--file-processor-allowed-ext requires --file-processor")
	}
	if processor == "" {
		return nil, errors.New("--file-processor must specify an executable")
	}
	if !extensionsSet || strings.TrimSpace(extensions) == "" {
		return nil, errors.New("--file-processor requires a non-empty --file-processor-allowed-ext")
	}
	allowed := strings.Split(extensions, ",")
	for i, extension := range allowed {
		allowed[i] = strings.TrimPrefix(strings.TrimSpace(extension), ".")
		if allowed[i] == "" {
			return nil, errors.New("empty extension in --file-processor-allowed-ext")
		}
	}
	return allowed, nil
}

func matchesExtension(path string, extensions []string) bool {
	for _, extension := range extensions {
		if extension == "*" || strings.HasSuffix(filepath.Base(path), "."+extension) {
			return true
		}
	}
	return false
}
