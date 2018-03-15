// Copyright 2018 CoreOS, Inc.
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

// Package errors includes errors that are used in multiple config versions
package errors

import (
	"errors"
	"fmt"
)

var (
	ErrInvalid            = errors.New("config is not valid")
	ErrCloudConfig        = errors.New("not a config (found coreos-cloudconfig)")
	ErrEmpty              = errors.New("not a config (empty)")
	ErrUnknownVersion     = errors.New("unsupported config version")
	ErrScript             = errors.New("not a config (found coreos-cloudinit script)")
	ErrDeprecated         = errors.New("config format deprecated")
	ErrVersion            = errors.New("incorrect config version")
	ErrCompressionInvalid = errors.New("invalid compression method")
)

// NewNoInstallSectionError produces an error indicating the given unit, named
// name, is missing an Install section.
func NewNoInstallSectionError(name string) error {
	return fmt.Errorf("unit %q is enabled, but has no install section so enable does nothing", name)
}
