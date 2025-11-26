// Copyright 2022 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"context"
	"fmt"
)

// Global registry for all translators.
// Variants register in init() functions.
var Global = NewRegistry()

type Registry struct {
	translators map[string]Translator
}

func NewRegistry() *Registry {
	return &Registry{
		translators: make(map[string]Translator),
	}
}

// Register adds a translator. Panics if already registered.
func (r *Registry) Register(t Translator) {
	meta := t.Metadata()
	key := meta.commonFields.asKey()

	if _, exists := r.translators[key]; exists {
		panic(fmt.Sprintf("translator already registered: %s version %s",
			meta.Variant, meta.Version.String()))
	}

	r.translators[key] = t
}

// Get retrieves a translator by variant and version.
func (r *Registry) Get(variant, version string) (Translator, error) {
	cf, err := newCF(variant, version)
	if err != nil {
		return nil, fmt.Errorf("invalid variant/version: %w", err)
	}

	key := cf.asKey()
	t, ok := r.translators[key]
	if !ok {
		return nil, fmt.Errorf("no translator registered for %s version %s", variant, version)
	}

	return t, nil
}

func (r *Registry) IsRegistered(variant, version string) bool {
	cf, err := newCF(variant, version)
	if err != nil {
		return false
	}

	_, ok := r.translators[cf.asKey()]
	return ok
}

// List returns all registered translator metadata.
func (r *Registry) List() []Metadata {
	result := make([]Metadata, 0, len(r.translators))
	for _, t := range r.translators {
		result = append(result, t.Metadata())
	}
	return result
}

// Translate auto-detects variant/version and translates the input.
func (r *Registry) Translate(ctx context.Context, input []byte, opts Options) (Result, error) {
	variant, version, err := ParseVariantVersion(input)
	if err != nil {
		return Result{}, fmt.Errorf("failed to parse variant/version: %w", err)
	}

	t, err := r.Get(variant, version)
	if err != nil {
		return Result{}, err
	}

	return t.Translate(ctx, input, opts)
}
