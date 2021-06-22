// Copyright 2020 Red Hat, Inc.
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

package types

import (
	"net/url"

	"github.com/coreos/ignition/v2/config/shared/errors"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func (res Resource) Key() string {
	for _, src := range res.GetSources() {
		return string(src)
	}
	return ""
}

func (res Resource) Validate(c path.ContextPath) (r report.Report) {
	r.AddOnError(c.Append("compression"), res.validateCompression())
	r.AddOnError(c.Append("verification", "hash"), res.validateVerification())
	r.AddOnError(c.Append("source"), res.validateSources())
	r.AddOnError(c.Append("httpHeaders"), res.validateSchemeForHTTPHeaders())
	return
}

func (res Resource) validateSources() error {
	if res.Source != nil && len(res.Sources) > 0 {
		return errors.ErrSourcesInvalid
	}

	for _, src := range res.GetSources() {
		if src == "" {
			continue
		}
		if err := validateURL(string(src)); err != nil {
			return err
		}
	}
	return nil
}

func (res Resource) validateCompression() error {
	if res.Compression != nil {
		switch *res.Compression {
		case "", "gzip":
		default:
			return errors.ErrCompressionInvalid
		}
	}
	return nil
}

func (res Resource) validateVerification() error {
	if res.Verification.Hash != nil && len(res.GetSources()) == 0 {
		return errors.ErrVerificationAndNilSource
	}
	return nil
}

func (res Resource) validateSchemeForHTTPHeaders() error {
	if len(res.HTTPHeaders) < 1 {
		return nil
	}

	sources := res.GetSources()
	if len(sources) == 0 {
		return errors.ErrInvalidUrl
	}

	for _, src := range sources {
		u, err := url.Parse(string(src))
		if err != nil {
			return errors.ErrInvalidUrl
		}

		switch u.Scheme {
		case "http", "https":
			continue
		default:
			return errors.ErrUnsupportedSchemeForHTTPHeaders
		}
	}
	return nil
}

// Ensure that the Source is specified and valid.  This is not called by
// Resource.Validate() because some structs that embed Resource don't
// require Source to be specified.  Containing structs that require Source
// should call this function from their Validate().
func (res Resource) validateRequiredSource() error {
	if err := res.validateSources(); err != nil {
		return err
	}
	if len(res.GetSources()) == 0 {
		return errors.ErrSourceRequired
	}
	return nil
}

func (res Resource) GetSources() []Source {
	if res.Source != nil {
		return []Source{Source(*res.Source)}
	}
	return res.Sources
}

func (res Resource) SourcesAreEmpty() bool {
	for _, src := range res.GetSources() {
		if src != "" {
			return false
		}
	}
	return true
}
