// Copyright 2019 Red Hat, Inc.
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

package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	v3_0 "github.com/coreos/ignition/v2/config/v3_0/types"
	v3_1 "github.com/coreos/ignition/v2/config/v3_1/types"
	v3_2 "github.com/coreos/ignition/v2/config/v3_2/types"
	v3_3 "github.com/coreos/ignition/v2/config/v3_3_experimental/types"
)

// helper to check whether a type and field matches a denylist of known problems
// examples are either structs or names of structs
func ignore(t reflect.Type, field reflect.StructField, fieldName string, examples ...interface{}) bool {
	if field.Name != fieldName {
		return false
	}
	for _, candidate := range examples {
		if reflect.TypeOf(candidate).Kind() == reflect.String {
			if t.Name() == candidate.(string) {
				return true
			}
		} else if t == reflect.TypeOf(candidate) {
			return true
		}
	}
	return false
}

// vary the specified field value and check the given key function to see
// whether the field seems to affect it
// this function's heuristic can be fooled by complex key functions but it
// should be fine for typical cases
func fieldAffectsKey(key func() string, v reflect.Value) bool {
	kind := v.Kind()
	switch {
	case util.IsPrimitive(kind):
		old := key()
		v.Set(util.NonZeroValue(v.Type()))
		new := key()
		v.Set(reflect.Zero(v.Type()))
		return old != new
	case kind == reflect.Ptr:
		null := key()
		v.Set(reflect.New(v.Type().Elem()))
		allocated := key()
		affectsKey := fieldAffectsKey(key, v.Elem())
		v.Set(reflect.Zero(v.Type()))
		return null != allocated || affectsKey
	case kind == reflect.Struct:
		ret := false
		for i := 0; i < v.NumField(); i++ {
			ret = ret || fieldAffectsKey(key, v.Field(i))
		}
		return ret
	case kind == reflect.Slice:
		if v.Len() > 0 {
			panic("Slice started with non-zero length")
		}
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		ret := fieldAffectsKey(key, v.Index(0))
		v.SetLen(0)
		return ret
	default:
		panic(fmt.Sprintf("Unexpected value kind %v", kind.String()))
	}
}

// check the fields that affect the key function of a keyed struct
// to ensure that we're using pointer and non-pointer fields properly.
func checkStructFieldKey(t reflect.Type) error {
	v := reflect.New(t).Elem()
	// wrapper to get the current key of @v
	getKey := func() string {
		// outer function's caller should have ensured that type
		// implements Keyed
		return v.Interface().(util.Keyed).Key()
	}

	var haveNonPointerKey bool
	// check the fields of one struct
	var checkStruct func(t reflect.Type, v reflect.Value) error
	checkStruct = func(t reflect.Type, v reflect.Value) error {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			affectsKey := fieldAffectsKey(getKey, v.Field(i))

			switch {
			case util.IsPrimitive(field.Type.Kind()):
				// non-pointer primitive; must affect key
				haveNonPointerKey = true
				if !affectsKey &&
					!ignore(t, field, "Target", v3_0.LinkEmbedded1{}, v3_1.LinkEmbedded1{}, v3_2.LinkEmbedded1{}, v3_3.LinkEmbedded1{}) &&
					!ignore(t, field, "Level", v3_0.Raid{}, v3_1.Raid{}, v3_2.Raid{}, v3_3.Raid{}) {
					return fmt.Errorf("Non-pointer %s.%s does not affect key", t.Name(), field.Name)
				}
			case field.Type.Kind() == reflect.Ptr && util.IsPrimitive(field.Type.Elem().Kind()):
				// pointer primitive; may affect key if there's also
				// a non-pointer key
			case field.Type.Kind() == reflect.Struct && field.Anonymous:
				// anonymous child struct; treat it as an extension of the
				// parent
				if err := checkStruct(field.Type, v.Field(i)); err != nil {
					return err
				}
			default:
				// slice, struct, or invalid type
				if affectsKey {
					return fmt.Errorf("Non-primitive %s.%s affects key", t.Name(), field.Name)
				}
			}
		}
		return nil
	}
	if err := checkStruct(t, v); err != nil {
		return err
	}

	// The Resource struct in spec >= 3.1 uses Source as the key, but
	// it's a pointer because in storage.files the source is optional.
	// Allow this special case, and the similar ConfigReference one in
	// 3.0.  This rule is a consistency guideline anyway; there's no
	// technical reason we can't have pointer keys.
	if !haveNonPointerKey &&
		t.Name() != "Resource" &&
		t != reflect.TypeOf(v3_0.ConfigReference{}) {
		return fmt.Errorf("No non-pointer key for %s", t.Name())
	}
	return nil
}

func testConfigType(t reflect.Type) error {
	k := t.Kind()
	switch {
	case util.IsInvalidInConfig(k):
		return fmt.Errorf("Type %s is of kind %s which is not valid in configs", t.Name(), k.String())
	case util.IsPrimitive(k):
		return nil
	case k == reflect.Ptr:
		pK := t.Elem().Kind()
		if util.IsPrimitive(pK) {
			return nil
		}
		switch t.Elem() {
		case reflect.TypeOf(v3_2.Clevis{}), reflect.TypeOf(v3_2.Custom{}):
			// these structs ended up with pointers; can't be helped now
			if err := testConfigType(t.Elem()); err != nil {
				return fmt.Errorf("Type %s has invalid children: %v", t.Elem().Name(), err)
			}
			return nil
		default:
			return fmt.Errorf("Type %s is a pointer that points to a non-primitive type", t.Name())
		}
	case k == reflect.Slice:
		eK := t.Elem().Kind()
		switch {
		case util.IsPrimitive(eK):
			return nil
		case eK == reflect.Struct:
			if err := testConfigType(t.Elem()); err != nil {
				return fmt.Errorf("Type %s has invalid children: %v", t.Name(), err)
			}
			return nil
		case eK == reflect.Slice:
			return fmt.Errorf("Type %s is a slice of slices", t.Name())
		case util.IsInvalidInConfig(eK):
			return fmt.Errorf("Type %s is a slice of invalid types", t.Name())
		default:
			return fmt.Errorf("Testing code encountered a failure at %s", t.Name())
		}
	case k == reflect.Struct:
		ignoredFields := map[string]struct{}{}
		if ignorer, ok := reflect.New(t).Interface().(util.IgnoresDups); ok {
			ignoredFields = ignorer.IgnoreDuplicates()
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if err := testConfigType(field.Type); err != nil {
				return fmt.Errorf("Type %s has invalid field %s: %v", t.Name(), field.Name, err)
			}
			if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() != reflect.String {
				elemType := field.Type.Elem()
				if _, ignored := ignoredFields[field.Name]; !ignored {
					// check this here, rather than in checkStructFieldKey(),
					// so we can provide more context in the error
					keyed, ok := reflect.New(elemType).Interface().(util.Keyed)
					if !ok {
						return fmt.Errorf("Type %s has slice field %s without Key() defined on %s debug: %v", t.Name(), field.Name, field.Type.Elem().Name(), ignoredFields)
					}
					// explicitly check for nil pointer dereference when calling Key() on zero value
					keyed.Key()
					if err := checkStructFieldKey(elemType); err != nil {
						return fmt.Errorf("Type %s has invalid field %s: %v", t.Name(), field.Name, err)
					}
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("Testing code encountered a failure at %s", t.Name())
	}
}

// TestConfigStructure walks the types of all our configs and ensures they don't contain
// anything the merge, translation, or validation logic doesn't know how to handle
func TestConfigStructure(t *testing.T) {
	configs := []reflect.Type{
		reflect.TypeOf(v3_0.Config{}),
		reflect.TypeOf(v3_1.Config{}),
		reflect.TypeOf(v3_2.Config{}),
		reflect.TypeOf(v3_3.Config{}),
	}

	for _, configType := range configs {
		if err := testConfigType(configType); err != nil {
			t.Errorf("Type %s/%s was invalid: %v", configType.PkgPath(), configType.Name(), err)
		}
	}
}
