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

package validate

import (
	"reflect"

	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/ajeddeloh/vcontext/json"
	"github.com/ajeddeloh/vcontext/path"
	"github.com/ajeddeloh/vcontext/report"
	"github.com/ajeddeloh/vcontext/validate"
)

func ValidateDups(v reflect.Value, c path.ContextPath) (r report.Report) {
	if v.Kind() != reflect.Struct {
		return
	}
	dupsLists := map[string]map[string]struct{}{}
	ignoreDups := map[string]struct{}{}
	if i, ok := v.Interface().(util.IgnoresDups); ok {
		ignoreDups = i.IgnoreDuplicates()
	}
	mergedKeys := map[string]string{}
	if m, ok := v.Interface().(util.MergesKeys); ok {
		mergedKeys = m.MergedKeys()
	}

	fields := validate.GetFields(v)
	for _, field := range fields {
		if field.Type.Kind() != reflect.Slice {
			continue
		}
		if _, ignored := ignoreDups[field.Name]; ignored {
			continue
		}
		dupListName := field.Name
		if mergedName, ok := mergedKeys[field.Name]; ok {
			dupListName = mergedName
		}
		dupList := dupsLists[dupListName]
		if dupList == nil {
			dupsLists[dupListName] = make(map[string]struct{}, field.Value.Len())
			dupList = dupsLists[dupListName]
		}
		for i := 0; i < field.Value.Len(); i++ {
			key := util.CallKey(field.Value.Index(i))
			if _, isDup := dupList[key]; isDup {
				r.AddOnError(c.Append(validate.FieldName(field, c.Tag), i), errors.ErrDuplicate)
			}
			dupList[key] = struct{}{}
		}
	}
	return
}

func ValidateWithContext(cfg interface{}, raw []byte) report.Report {
	r := validate.Validate(cfg, "json")
	r.Merge(validate.ValidateCustom(cfg, "json", ValidateDups))

	if cxt, err := json.UnmarshalToContext(raw); err == nil {
		r.Correlate(cxt)
	}
	return r
}
