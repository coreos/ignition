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
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
	"github.com/stretchr/testify/assert"
)

var (
	ErrB   = fmt.Errorf("err B")
	ErrI   = fmt.Errorf("err I")
	ErrS   = fmt.Errorf("err S")
	ErrSB  = fmt.Errorf("err SB")
	ErrSSB = fmt.Errorf("err SSB")
	ErrBP  = fmt.Errorf("err BP")
	ErrIP  = fmt.Errorf("err IP")
	ErrSP  = fmt.Errorf("err SP")
	ErrF   = fmt.Errorf("err F")
)

type StrA struct {
	B   bool   `json:"b"`
	I   int    `json:"i"`
	S   string `json:"s"`
	SB  StrB   `json:"sb"`
	SSB []StrB `json:"ssb"`
}

type StrB struct {
	BP *bool   `json:"bp"`
	IP *int    `json:"ip"`
	SP *string `json:"sp"`
	StrC
}

type StrC struct {
	F float64 `json:"f"`
}

func TestNewFilters(t *testing.T) {
	NewFilters(StrA{}, FilterMap{
		"b":      ErrB,
		"i":      ErrI,
		"s":      ErrS,
		"sb.bp":  ErrBP,
		"ssb.ip": ErrIP,
		"sb.f":   ErrF,
	})
	assert.Panics(t, func() {
		NewFilters(StrA{}, FilterMap{
			"z": ErrB,
		})
	})
	assert.Panics(t, func() {
		NewFilters(StrA{}, FilterMap{
			"ssb.z": ErrB,
		})
	})
	assert.Panics(t, func() {
		NewFilters(StrA{}, FilterMap{
			"ssb.ip.z": ErrB,
		})
	})
	assert.Panics(t, func() {
		NewFilters(StrA{}, FilterMap{
			"sb.f.z": ErrB,
		})
	})
}

func TestFilter(t *testing.T) {
	obj := StrA{
		I: 7,
		S: "hello",
		SB: StrB{
			BP: util.BoolToPtr(true),
			IP: util.IntToPtr(7),
			SP: util.StrToPtr("goodbye"),
			StrC: StrC{
				F: 3.1,
			},
		},
		SSB: []StrB{
			{
				BP: util.BoolToPtr(true),
			},
			{
				SP: util.StrToPtr("str"),
			},
			{
				SP: util.StrToPtr(""),
			},
		},
	}

	// no filters, no errors
	assert.Equal(t, report.Report{}, NewFilters(StrA{}, FilterMap{}).Verify(obj))

	// various filters, ignore zero
	var expected report.Report
	expected.AddOnError(path.New("json", "sb", "ip"), ErrIP)
	expected.AddOnError(path.New("json", "sb", "f"), ErrF)
	expected.AddOnError(path.New("json", "ssb", 0, "bp"), ErrBP)
	expected.AddOnError(path.New("json", "ssb", 1, "sp"), ErrSP)
	assert.Equal(t, expected, NewFiltersIgnoreZero(StrA{}, FilterMap{
		"b":      ErrB,
		"sb.ip":  ErrIP,
		"sb.f":   ErrF,
		"ssb.bp": ErrBP,
		"ssb.ip": ErrIP,
		"ssb.sp": ErrSP,
	}, []string{
		"ssb.sp",
	}).Verify(obj))
	// stop ignoring zero
	expected.AddOnError(path.New("json", "ssb", 2, "sp"), ErrSP)
	assert.Equal(t, expected, NewFilters(StrA{}, FilterMap{
		"b":      ErrB,
		"sb.ip":  ErrIP,
		"sb.f":   ErrF,
		"ssb.bp": ErrBP,
		"ssb.ip": ErrIP,
		"ssb.sp": ErrSP,
	}).Verify(obj))

	// filter stops descent
	expected = report.Report{}
	expected.AddOnError(path.New("json", "ssb"), ErrSSB)
	assert.Equal(t, expected, NewFilters(StrA{}, FilterMap{
		"ssb":    ErrSSB,
		"ssb.bp": ErrBP,
		"ssb.ip": ErrIP,
		"ssb.sp": ErrSP,
	}).Verify(obj))
}
