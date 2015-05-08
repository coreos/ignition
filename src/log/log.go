// Copyright 2015 CoreOS, Inc.
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
	"log/syslog"
	"strings"
)

type LoggerOps interface {
	Emerg(string) error
	Alert(string) error
	Crit(string) error
	Err(string) error
	Warning(string) error
	Notice(string) error
	Info(string) error
	Debug(string) error
	Close() error
}

// Logger implements a variadic flavor of log/syslog.Writer
type Logger struct {
	ops           LoggerOps
	prefixStack   []string
	opSequenceNum int
}

func New() Logger {
	logger := Logger{}
	if slogger, err := syslog.New(syslog.LOG_DEBUG, "ignition"); err == nil {
		logger.ops = slogger
	} else {
		logger.ops = Stdout{}
		logger.Err("unable to open syslog: %v", err)
	}
	return logger
}

func (l Logger) Close() {
	l.ops.Close()
}

func (l Logger) Emerg(format string, a ...interface{}) error {
	return l.log(l.ops.Emerg, format, a...)
}

func (l Logger) Alert(format string, a ...interface{}) error {
	return l.log(l.ops.Alert, format, a...)
}

func (l Logger) Crit(format string, a ...interface{}) error {
	return l.log(l.ops.Crit, format, a...)
}

func (l Logger) Err(format string, a ...interface{}) error {
	return l.log(l.ops.Err, format, a...)
}

func (l Logger) Warning(format string, a ...interface{}) error {
	return l.log(l.ops.Warning, format, a...)
}

func (l Logger) Notice(format string, a ...interface{}) error {
	return l.log(l.ops.Notice, format, a...)
}

func (l Logger) Info(format string, a ...interface{}) error {
	return l.log(l.ops.Info, format, a...)
}

func (l Logger) Debug(format string, a ...interface{}) error {
	return l.log(l.ops.Debug, format, a...)
}

func (l *Logger) PushPrefix(format string, a ...interface{}) {
	l.prefixStack = append(l.prefixStack, fmt.Sprintf(format, a...))
}

func (l *Logger) PopPrefix() {
	if len(l.prefixStack) == 0 {
		l.Debug("popped from empty stack")
		return
	}
	l.prefixStack = l.prefixStack[:len(l.prefixStack)-1]
}

func (l *Logger) LogOp(op func() error, format string, a ...interface{}) error {
	l.opSequenceNum++
	l.PushPrefix("op(%x)", l.opSequenceNum)
	defer l.PopPrefix()

	l.logStart(format, a...)
	if err := op(); err != nil {
		l.logFail("%s: %v", fmt.Sprintf(format, a...), err)
		return err
	}
	l.logFinish(format, a...)
	return nil
}

func (l Logger) logStart(format string, a ...interface{}) {
	l.Info(fmt.Sprintf("[started]  %s", format), a...)
}

func (l Logger) logFail(format string, a ...interface{}) {
	l.Crit(fmt.Sprintf("[failed]   %s", format), a...)
}

func (l Logger) logFinish(format string, a ...interface{}) {
	l.Info(fmt.Sprintf("[finished] %s", format), a...)
}

func (l Logger) log(logFunc func(string) error, format string, a ...interface{}) error {
	return logFunc(l.sprintf(format, a...))
}

func (l Logger) sprintf(format string, a ...interface{}) string {
	m := []string{}
	for _, pfx := range l.prefixStack {
		m = append(m, fmt.Sprintf("%s:", pfx))
	}
	m = append(m, fmt.Sprintf(format, a...))
	return strings.Join(m, " ")
}
