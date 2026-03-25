// Copyright 2021 Red Hat, Inc.
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

package translate

import (
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	old "github.com/coreos/ignition/v2/config/v3_6/types"
)

// Check that we have valid translators for the complete config struct
// hierarchy; Translate will panic if not.  We need to use a deeply non-zero
// struct to ensure translation descends into every type.
func TestTranslate(t *testing.T) {
	typ := reflect.TypeOf(old.Config{})
	config := util.NonZeroValue(typ).Interface().(old.Config)
	Translate(config)
}
