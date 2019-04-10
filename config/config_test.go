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
)

func testConfigType(t reflect.Type) error {
	k := t.Kind()
	switch {
	case util.IsInvalidInConfig(k):
		return fmt.Errorf("Type %s is of kind %s which is not valid in configs", t.Name(), k.String())
	case util.IsPrimitive(k):
		return nil
	case k == reflect.Ptr:
		pK := t.Elem().Kind()
		switch {
		case util.IsPrimitive(pK):
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
				if _, ignored := ignoredFields[field.Name]; !ignored {
					if _, ok := reflect.New(field.Type.Elem()).Interface().(util.Keyed); !ok {
						return fmt.Errorf("Type %s has slice field %s without Key() defined on %s debug: %v", t.Name(), field.Name, field.Type.Elem().Name(), ignoredFields)
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
	}

	for _, configType := range configs {
		if err := testConfigType(configType); err != nil {
			t.Errorf("Type %s was invalid: %v", configType.Name(), err)
		}
	}
}
