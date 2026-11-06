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

type multiOps []LoggerOps

func (m multiOps) each(fn func(LoggerOps) error) error {
	var err error
	for _, o := range m {
		if e := fn(o); e != nil {
			err = e
		}
	}
	return err
}

func (m multiOps) Emerg(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Emerg(msg) })
}

func (m multiOps) Alert(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Alert(msg) })
}

func (m multiOps) Crit(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Crit(msg) })
}

func (m multiOps) Err(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Err(msg) })
}

func (m multiOps) Warning(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Warning(msg) })
}

func (m multiOps) Notice(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Notice(msg) })
}

func (m multiOps) Info(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Info(msg) })
}

func (m multiOps) Debug(msg string) error {
	return m.each(func(o LoggerOps) error { return o.Debug(msg) })
}

func (m multiOps) Close() error {
	return m.each(func(o LoggerOps) error { return o.Close() })
}
