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

package ignconverter

func strP(in string) *string {
	if in == "" {
		return nil
	}
	return &in
}

func strPStrict(in string) *string {
	return &in
}

func boolP(in bool) *bool {
	if !in {
		return nil
	}
	return &in
}

func boolPStrict(in bool) *bool {
	return &in
}

func intP(in int) *int {
	if in == 0 {
		return nil
	}
	return &in
}

func strV(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}

func boolV(in *bool) bool {
	if in == nil {
		return false
	}
	return *in
}

func intV(in *int) int {
	if in == nil {
		return 0
	}
	return *in
}
