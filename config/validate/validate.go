// Copyright 2015 CoreOS, Inc.
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

package validate

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	json "github.com/ajeddeloh/go-json"
	"github.com/coreos/ignition/config/types"
	"github.com/coreos/ignition/config/validate/report"
	"go4.org/errorutil"
)

type assertValidator interface {
	AssertValid() error
}

// wrapper for errorutil that handles missing sources sanely and resets the reader afterwards
func posFromOffset(offset int, source io.ReadSeeker) (int, int, string) {
	if source == nil {
		return 0, 0, ""
	}
	line, col, highlight := errorutil.HighlightBytePosition(source, int64(offset))
	source.Seek(0, 0) // Reset the reader to the start so the next call isn't relative to this position
	return line, col, highlight
}

func Validate(cfg types.Config, ast json.Node, source io.ReadSeeker) report.Report {
	v := reflect.ValueOf(cfg)
	r := validate(v, ast, source)
	r.Merge(applyRules(cfg))
	return r
}

// Validate walks down a struct tree calling AssertValid on every node that implements it, building
// A report of all the errors, warnings, info, and deprecations it encounters
func validate(vObj reflect.Value, ast json.Node, source io.ReadSeeker) (r report.Report) {
	if !vObj.IsValid() {
		return
	}

	line, col, highlight := posFromOffset(ast.End, source)

	// See if we A) can call AssertValid on vObj, and B) should call AssertValid. AssertValid should NOT be called
	// when vObj is nil, as it will panic or when vObj is a pointer to a value with AssertValid implemented with a
	// value receiver. This is to prevent AssertValid being called twice, as otherwise it would be called on the
	// pointer version (due to go's automatic deferencing) and once when the pointer is deferenced below. The only
	// time AssertValid should be called on a pointer is when the function is implemented with a pointer reciever.
	if obj, ok := vObj.Interface().(assertValidator); ok &&
		((vObj.Kind() != reflect.Ptr) ||
			(!vObj.IsNil() && !vObj.Elem().Type().Implements(reflect.TypeOf((*assertValidator)(nil)).Elem()))) {
		if err := obj.AssertValid(); err != nil {
			r.Add(report.Entry{
				Kind:      report.EntryError,
				Message:   err.Error(),
				Line:      line,
				Column:    col,
				Highlight: highlight,
			})
			// Dont recurse on invalid inner nodes, it mostly leads to bogus messages
			return
		}
	}

	switch vObj.Kind() {
	case reflect.Ptr:
		sub_report := validate(vObj.Elem(), ast, source)
		sub_report.AddPosition(line, col)
		r.Merge(sub_report)
	case reflect.Struct:
		sub_report := validateStruct(vObj, ast, source)
		sub_report.AddPosition(line, col)
		r.Merge(sub_report)
	case reflect.Slice:
		for i := 0; i < vObj.Len(); i++ {
			sub_node := ast
			if val, ok := ast.Value.([]json.Node); ok {
				sub_node = val[i]
			}
			sub_report := validate(vObj.Index(i), sub_node, source)
			sub_report.AddPosition(line, col)
			r.Merge(sub_report)
		}
	}
	return
}

func ValidateWithoutSource(cfg types.Config) (report report.Report) {
	return Validate(cfg, json.Node{}, nil)
}

func validateStruct(vObj reflect.Value, ast json.Node, source io.ReadSeeker) report.Report {
	off := ast.End
	r := report.Report{}

	// isFromObject will be true if this struct was unmarshalled from a JSON object.
	keys, isFromObject := ast.Value.(map[string]json.Node)

	// Maintain a set of key's that have been used.
	usedKeys := map[string]struct{}{}

	// Maintain a list of all the tags in the struct for fuzzy matching later.
	tags := []string{}

	for i := 0; i < vObj.Type().NumField(); i++ {
		// Default to zero value json.Node if the field's corrosponding node cannot be found.
		var sub_node json.Node
		// Default to passing a nil source if the field's corrosponding node cannot be found.
		// This ensures the line numbers reported from all sub-structs are 0 and will be changed by AddPosition
		var src io.ReadSeeker

		// Try to determine the json.Node that corrosponds with the struct field
		if isFromObject {
			tag := strings.SplitN(vObj.Type().Field(i).Tag.Get("json"), ",", 2)[0]
			// Save the tag so we have a list of all the tags in the struct
			tags = append(tags, tag)
			// mark that this key was used
			usedKeys[tag] = struct{}{}

			if sub, ok := keys[tag]; ok {
				// Found it
				sub_node = sub
				src = source
			}
		}
		sub_report := validate(vObj.Field(i), sub_node, src)
		// Default to deepest node if the node's type isn't an object,
		// such as when a json string actually unmarshal to structs (like with version)
		line, col, _ := posFromOffset(off, src)
		sub_report.AddPosition(line, col)
		r.Merge(sub_report)
	}
	if !isFromObject {
		// If this struct was not unmarshalled from a JSON object, there cannot be unused keys.
		return r
	}

	for k, v := range keys {
		if _, hasKey := usedKeys[k]; hasKey {
			continue
		}
		line, col, highlight := posFromOffset(v.KeyEnd, source)
		typo := similar(k, tags)

		r.Add(report.Entry{
			Kind:      report.EntryWarning,
			Message:   fmt.Sprintf("Config has unrecognized key: %s", k),
			Line:      line,
			Column:    col,
			Highlight: highlight,
		})

		if typo != "" {
			r.Add(report.Entry{
				Kind:      report.EntryInfo,
				Message:   fmt.Sprintf("Did you mean %s instead of %s", typo, k),
				Line:      line,
				Column:    col,
				Highlight: highlight,
			})
		}
	}

	return r
}

// similar returns a string in candidates that is similar to str. Currently it just does case
// insensitive comparison, but it should be updated to use levenstein distances to catch typos
func similar(str string, candidates []string) string {
	for _, candidate := range candidates {
		if strings.EqualFold(str, candidate) {
			return candidate
		}
	}
	return ""
}
