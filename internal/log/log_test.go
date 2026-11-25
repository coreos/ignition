// Copyright 2026 CoreOS, Inc.
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

package log

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recorder struct {
	calls  []string
	closed int
	fail   error
}

func (r *recorder) log(level, msg string) error {
	r.calls = append(r.calls, level+":"+msg)
	return r.fail
}

func (r *recorder) Emerg(msg string) error   { return r.log("emerg", msg) }
func (r *recorder) Alert(msg string) error   { return r.log("alert", msg) }
func (r *recorder) Crit(msg string) error    { return r.log("crit", msg) }
func (r *recorder) Err(msg string) error     { return r.log("err", msg) }
func (r *recorder) Warning(msg string) error { return r.log("warning", msg) }
func (r *recorder) Notice(msg string) error  { return r.log("notice", msg) }
func (r *recorder) Info(msg string) error    { return r.log("info", msg) }
func (r *recorder) Debug(msg string) error   { return r.log("debug", msg) }
func (r *recorder) Close() error {
	r.closed++
	return r.fail
}

func TestLoggerInfoUsesOps(t *testing.T) {
	sw := &recorder{}
	l := Logger{ops: sw}
	l.Info("hello %s", "world")
	assert.Equal(t, []string{"info:hello world"}, sw.calls)
}

func TestMultiOpsFansOut(t *testing.T) {
	a := &recorder{}
	b := &recorder{}
	m := multiOps{a, b}

	assert.NoError(t, m.Emerg("e"))
	assert.NoError(t, m.Alert("a"))
	assert.NoError(t, m.Crit("c"))
	assert.NoError(t, m.Err("r"))
	assert.NoError(t, m.Warning("w"))
	assert.NoError(t, m.Notice("n"))
	assert.NoError(t, m.Info("i"))
	assert.NoError(t, m.Debug("d"))

	want := []string{
		"emerg:e", "alert:a", "crit:c", "err:r",
		"warning:w", "notice:n", "info:i", "debug:d",
	}
	assert.Equal(t, want, a.calls)
	assert.Equal(t, want, b.calls)
}

func TestMultiOpsReturnsLastError(t *testing.T) {
	a := &recorder{fail: fmt.Errorf("first")}
	b := &recorder{fail: fmt.Errorf("second")}
	m := multiOps{a, b}
	assert.EqualError(t, m.Info("x"), "second")
	assert.Equal(t, []string{"info:x"}, a.calls)
	assert.Equal(t, []string{"info:x"}, b.calls)
}

func TestMultiOpsClose(t *testing.T) {
	a := &recorder{}
	b := &recorder{fail: fmt.Errorf("close b")}
	m := multiOps{a, b}
	assert.EqualError(t, m.Close(), "close b")
	assert.Equal(t, 1, a.closed)
	assert.Equal(t, 1, b.closed)
}
