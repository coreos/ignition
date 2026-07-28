// Copyright 2020 Red Hat, Inc
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

// Package report allows butane to pretty print errors, the error format is as follows:
package report

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/coreos/vcontext/report"
)

const (
	red    = "\033[1;31m"
	yellow = "\033[1;33m"
	cyan   = "\033[1;36m"
	blue   = "\033[1;34m"
	reset  = "\033[0m"
)

func FormatError(r report.Report, fileName string, source []byte, colorize, rawErrors bool) string {
	if rawErrors {
		return formatErrorSimple(r)
	} else {
		return formatErrorPretty(r, fileName, source, colorize)
	}
}

func formatErrorSimple(r report.Report) string {
	return r.String()
}

func formatErrorPretty(r report.Report, fileName string, source []byte, colorize bool) string {
	lines := strings.Split(string(source), "\n")
	var buf strings.Builder
	for i, entry := range r.Entries {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(formatErrorEntry(entry, fileName, lines, colorize))
	}
	return buf.String()
}

func color(text, code string, colorize bool) string {
	if !colorize {
		return text
	}
	return code + text + reset
}

func severityColor(kind report.EntryKind) string {
	switch kind {
	case report.Error:
		return red
	case report.Info:
		return cyan
	case report.Warn:
		return yellow
	default:
		return reset
	}
}

func writeUnderline(buf *strings.Builder, col, gutterWidth int, message, line string, colorize bool) {
	underlineStart := col - 1
	rest := line[underlineStart:]
	nextWhitespace := strings.IndexFunc(rest, unicode.IsSpace)
	if nextWhitespace == -1 {
		// If we didn't find whitespace then that means that we need to underline the entire string
		nextWhitespace = len(rest)
	}
	underlineEnd := underlineStart + nextWhitespace
	padding := strings.Repeat(" ", underlineStart)
	underline := strings.Repeat("^", underlineEnd-underlineStart)
	underline = color(underline, blue, colorize)

	fmt.Fprintf(buf, " %s | %s%s %s\n",
		strings.Repeat(" ", gutterWidth), padding, underline, message)
}

// formatErrorEntry will try to return the error as a pretty string in the following form
//
// error[$.boot_device.layout]:
//
//	 --> ../testing.bu:4:11
//	  |
//	2 | version: 1.6.0
//	3 | boot_device:
//	4 |   layout: s390x-virt
//	  |           ^^^^^^^^^^ mirroring not supported on layouts: s390x-eckd, s390x-zfcp, s390x-virt
//	5 |   mirror:
//	6 |     devices:
//	  |
func formatErrorEntry(entry report.Entry, filename string, lines []string, colorize bool) string {
	if entry.Marker.StartP == nil {
		return entry.String() + "\n"
	}

	line := int(entry.Marker.StartP.Line)
	col := int(entry.Marker.StartP.Column)

	// this should never happen as lines and cols are 1 indexed, but we'll add a check in case the vcontext library ever changes
	if line < 1 || line > len(lines) || col < 1 || col > len(lines[line-1]) {
		return entry.String() + "\n"
	}

	var buf strings.Builder
	kindColor := severityColor(entry.Kind)
	kindStr := color(entry.Kind.String(), kindColor, colorize)

	path := ""
	if entry.Context.Len() > 0 {
		path = "[" + entry.Context.String() + "]"
	}
	fmt.Fprintf(&buf, "%s%s:\n", kindStr, path)
	// Add information about the location of the error in the following form:
	//
	// "  --> testing.bu:10:4"
	arrow := color("-->", blue, colorize)
	fmt.Fprintf(&buf, "  %s %s:%d:%d\n", arrow, filename, line, col)

	// Number of lines to show before and after the error location
	contextLines := 2
	contextLineInit := max(1, line-contextLines)
	contextLineEnd := min(len(lines), line+contextLines)

	// width of the largest line number
	gutterWidth := len(fmt.Sprintf("%d", contextLineEnd))
	fmt.Fprintf(&buf, " %s |\n", strings.Repeat(" ", gutterWidth))
	for lineNumber := contextLineInit; lineNumber <= contextLineEnd; lineNumber++ {
		lineNum := color(fmt.Sprintf("%*d", gutterWidth, lineNumber), blue, colorize)

		fmt.Fprintf(&buf, " %s | %s\n", lineNum, lines[lineNumber-1])
		// Underline the error and write the error message
		if lineNumber == line {
			writeUnderline(&buf, col, gutterWidth, entry.Message, lines[lineNumber-1], colorize)
		}
	}

	// Empty line at the end
	fmt.Fprintf(&buf, " %s |\n", strings.Repeat(" ", gutterWidth))
	return buf.String()
}
