// Copyright 2024 CoreOS, Inc.
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

package util_test

import (
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/internal/providers/util"
)

func TestTryUnzip(t *testing.T) {
	type args struct {
		raw []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "uncompressed",
			args: args{
				raw: []byte("hello world"),
			},
			want:    []byte("hello world"),
			wantErr: false,
		},
		{
			name: "compressed",
			args: args{
				raw: []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xcb, 0x48, 0xcd, 0xc9, 0xc9, 0x57, 0x28, 0xcf, 0x2f, 0xca, 0x49, 0x01, 0x00, 0x85, 0x11, 0x4a, 0x0d, 0x0b, 0x00, 0x00, 0x00},
			},
			want:    []byte("hello world"),
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				raw: []byte{0x1f, 0x8b, 0x08},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := util.TryUnzip(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("TryUnzip() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TryUnzip() = %v, want %v", got, tt.want)
			}
		})
	}
}
