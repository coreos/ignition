// Copyright 2023 Red Hat, Inc
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
// limitations under the License.)

package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

type FilterMap map[string]error

type FieldFilters struct {
	filters FilterMap
	// openshift 4.8 and 4.9 specs want to filter out the compression
	// field but ignore a pointer to a zero value (StrToPtr(""))
	// because those are generated automatically by desugaring.  Provide
	// a way to do that.
	ignoreZero map[string]struct{}
}

func NewFilters(v any, filters FilterMap) FieldFilters {
	return NewFiltersIgnoreZero(v, filters, []string{})
}

func NewFiltersIgnoreZero(v any, filters FilterMap, ignoreZero []string) FieldFilters {
	for filter := range filters {
		if !isValidFilter(reflect.TypeOf(v), filter) {
			panic(fmt.Errorf("invalid filter path: %s", filter))
		}
	}
	ignore := make(map[string]struct{})
	for _, value := range ignoreZero {
		ignore[value] = struct{}{}
	}
	return FieldFilters{
		filters:    filters,
		ignoreZero: ignore,
	}
}

func isValidFilter(typ reflect.Type, filter string) bool {
	if filter == "" {
		return true
	}
	kind := typ.Kind()
	switch {
	case util.IsPrimitive(kind):
		// can't descend further
		return false
	case kind == reflect.Struct:
		element, rest, _ := strings.Cut(filter, ".")
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.Anonymous {
				if isValidFilter(field.Type, filter) {
					return true
				}
			} else {
				if getTag(field) == element {
					return isValidFilter(field.Type, rest)
				}
			}
		}
		return false
	case kind == reflect.Slice, kind == reflect.Ptr:
		return isValidFilter(typ.Elem(), filter)
	default:
		panic(fmt.Errorf("%v has kind %v", typ.Name(), kind))
	}
}

func (ff FieldFilters) Verify(v any) report.Report {
	return ff.verify(reflect.ValueOf(v), "", path.New("json"))
}

func (ff FieldFilters) verify(v reflect.Value, filter string, p path.ContextPath) (r report.Report) {
	if err := ff.Lookup(filter); err != nil {
		// This object is filtered.  Add an error if it's non-empty,
		// but don't descend further in any case.
		if !ff.isEmpty(v, filter) {
			r.AddOnError(p, err)
		}
		return
	}

	typ := v.Type()
	kind := typ.Kind()
	switch {
	case util.IsPrimitive(kind):
	case kind == reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if typ.Field(i).Anonymous {
				r.Merge(ff.verify(field, filter, p))
			} else {
				tag := getTag(typ.Field(i))
				r.Merge(ff.verify(field, fmt.Sprintf("%s.%s", filter, tag), p.Append(tag)))
			}
		}
	case kind == reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			r.Merge(ff.verify(v.Index(i), filter, p.Append(i)))
		}
	case kind == reflect.Ptr:
		if !v.IsNil() {
			r.Merge(ff.verify(v.Elem(), filter, p))
		}
	case kind == reflect.Map:
		// not supported in filters; ignore
	default:
		panic(fmt.Errorf("%v has kind %v", typ.Name(), kind))
	}
	return
}

func (ff FieldFilters) isEmpty(v reflect.Value, filter string) bool {
	typ := v.Type()
	kind := typ.Kind()
	switch {
	case util.IsPrimitive(kind):
		return v.IsZero()
	case kind == reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			childFilter := filter
			if !typ.Field(i).Anonymous {
				childFilter = fmt.Sprintf("%s.%s", filter, getTag(typ.Field(i)))
			}
			if !ff.isEmpty(v.Field(i), childFilter) {
				return false
			}
		}
		return true
	case kind == reflect.Slice:
		// different from v.IsZero(): we treat a non-nil zero-length
		// slice as empty
		return v.Len() == 0
	case kind == reflect.Ptr:
		if v.IsNil() {
			return true
		}
		// special case: if pointing to a primitive, and the primitive
		// is the zero value, and the filter is listed in ff.ignoreZero,
		// treat as empty
		if util.IsPrimitive(typ.Elem().Kind()) && v.Elem().IsZero() {
			_, ignoreZero := ff.ignoreZero[strings.TrimPrefix(filter, ".")]
			if ignoreZero {
				return true
			}
		}
		return false
	case kind == reflect.Map:
		// not supported in filters; prune
		return true
	default:
		panic(fmt.Errorf("%v has kind %v", typ.Name(), kind))
	}
}

func (ff FieldFilters) Lookup(filter string) error {
	return ff.filters[strings.TrimPrefix(filter, ".")]
}

func getTag(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		panic(fmt.Errorf("struct field %q has no JSON tag", field.Name))
	}
	return strings.Split(tag, ",")[0]
}
