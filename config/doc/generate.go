// Copyright 2023 Red Hat, Inc.
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

package doc

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/coreos/ignition/v2/config/util"
)

func descendNode(ver semver.Version, node DocNode, typ reflect.Type, level int, w io.Writer) error {
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct: %v (%v)", typ.Name(), typ.Kind())
	}
	fieldsByTag, err := structFieldsByTag(typ)
	if err != nil {
		return err
	}
	// iterate in order of docs YAML
	for _, child := range node.Children {
		field, ok := fieldsByTag[child.Name]
		if !ok {
			// have documentation but no struct field
			continue
		}
		var optional string
		if !util.IsTrue(child.Required) && (util.IsFalse(child.Required) || !util.IsPrimitive(field.Type.Kind())) {
			optional = "_"
		}
		// write the entry
		desc, err := child.renderDescription(ver)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s* **%s%s%s** (%s): %s\n", strings.Repeat("  ", level), optional, child.Name, optional, typeName(field.Type), desc); err != nil {
			return err
		}
		// recurse
		if err := descend(ver, child, field.Type, level+1, w); err != nil {
			return err
		}
		// delete from map to keep track of fields we've seen
		delete(fieldsByTag, child.Name)
	}
	// check for undocumented fields
	for _, field := range fieldsByTag {
		return fmt.Errorf("undocumented field: %v.%v", typ.Name(), field.Name)
	}
	return nil
}

func descend(ver semver.Version, node DocNode, typ reflect.Type, level int, w io.Writer) error {
	kind := typ.Kind()
	switch {
	case util.IsPrimitive(kind):
		return nil
	case kind == reflect.Struct:
		return descendNode(ver, node, typ, level, w)
	case kind == reflect.Slice, kind == reflect.Ptr:
		return descend(ver, node, typ.Elem(), level, w)
	case kind == reflect.Map:
		if !util.IsPrimitive(typ.Key().Kind()) {
			return fmt.Errorf("%v is map with non-primitive key type %v", typ.Name(), typ.Key())
		}
		return descend(ver, node, typ.Elem(), level, w)
	default:
		return fmt.Errorf("%v has kind %v", typ.Name(), kind)
	}
}

func structFieldsByTag(typ reflect.Type) (map[string]reflect.StructField, error) {
	ret := make(map[string]reflect.StructField, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			// anonymous embedded structure; merge its fields
			sub, err := structFieldsByTag(field.Type)
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				ret[k] = v
			}
		} else {
			tag, ok := field.Tag.Lookup("yaml")
			if !ok {
				tag, ok = field.Tag.Lookup("json")
			}
			if !ok {
				return nil, fmt.Errorf("no field tag: %v.%v", typ.Name(), field.Name)
			}
			// extract the field name, ignoring omitempty etc.
			tag = strings.Split(tag, ",")[0]
			ret[tag] = field
		}
	}
	return ret, nil
}

func typeName(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int:
		return "integer"
	case reflect.Map:
		return "object"
	case reflect.Pointer:
		return typeName(typ.Elem())
	case reflect.Slice:
		return fmt.Sprintf("list of %ss", typeName(typ.Elem()))
	case reflect.String:
		return "string"
	case reflect.Struct:
		return "object"
	default:
		panic(fmt.Errorf("unknown type kind: %v", typ.Kind()))
	}
}
