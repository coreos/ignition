// Copyright 2019 Red Hat, Inc
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

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	breport "github.com/coreos/butane/internal/report"
	"github.com/coreos/butane/internal/version"
)

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func isCharDevice(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func main() {
	var (
		input       string
		output      string
		colorFlag   string
		check       bool
		strict      bool
		helpFlag    bool
		versionFlag bool
		rawErrors   bool
		colorize    bool
	)
	options := common.TranslateBytesOptions{}
	pflag.BoolVarP(&helpFlag, "help", "h", false, "show usage and exit")
	pflag.BoolVarP(&versionFlag, "version", "V", false, "print the version and exit")
	pflag.BoolVarP(&options.DebugPrintTranslations, "debug", "D", false, "log translations")
	pflag.Lookup("debug").Hidden = true
	pflag.BoolVarP(&check, "check", "c", false, "check config without producing output")
	pflag.BoolVarP(&strict, "strict", "s", false, "fail on any warning")
	pflag.BoolVarP(&options.Pretty, "pretty", "p", false, "output formatted json")
	pflag.BoolVarP(&options.Raw, "raw", "r", false, "never wrap in a MachineConfig; force Ignition output")
	pflag.BoolVar(&rawErrors, "raw-errors", false, "show raw errors, rather than pretty printing them")
	pflag.StringVar(&colorFlag, "color", "auto", `control color output: "auto", "always", or "never"`)
	pflag.Lookup("color").NoOptDefVal = "always"
	pflag.StringVar(&colorFlag, "colour", "auto", `control color output: "auto", "always", or "never"`)
	pflag.Lookup("colour").NoOptDefVal = "always"
	pflag.Lookup("colour").Hidden = true
	pflag.StringVar(&input, "input", "", "read from input file instead of stdin")
	pflag.Lookup("input").Deprecated = "specify filename directly on command line"
	pflag.Lookup("input").Hidden = true
	pflag.StringVarP(&output, "output", "o", "", "write to output file instead of stdout")
	pflag.StringVarP(&options.FilesDir, "files-dir", "d", "", "allow embedding local files from this directory")

	pflag.Usage = func() {
		fmt.Fprintf(pflag.CommandLine.Output(), "Usage: %s [options] [input-file]\n", os.Args[0])
		fmt.Fprintf(pflag.CommandLine.Output(), "Options:\n")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	switch colorFlag {
	case "always", "yes":
		colorize = true
	case "never", "no":
		colorize = false
	case "auto":
		_, noColorSet := os.LookupEnv("NO_COLOR")
		isTTY := isCharDevice(os.Stderr)
		colorize = !noColorSet && isTTY
	}

	args := pflag.Args()
	if len(args) == 1 && input == "" {
		input = args[0]
	} else if len(args) > 0 {
		pflag.Usage()
		os.Exit(2)
	}

	if helpFlag {
		pflag.CommandLine.SetOutput(os.Stdout)
		pflag.Usage()
		os.Exit(0)
	}

	if versionFlag {
		fmt.Println(version.String)
		os.Exit(0)
	}

	infile := os.Stdin
	filename := "<stdin>"
	if input != "" {
		var err error
		infile, err = os.Open(input)
		if err != nil {
			fail("failed to open %s: %v\n", input, err)
		}
		defer infile.Close()
		filename = input
	}

	dataIn, err := io.ReadAll(infile)
	if err != nil {
		fail("failed to read %s: %v\n", infile.Name(), err)
	}

	dataOut, r, err := config.TranslateBytes(dataIn, options)

	errorString := breport.FormatError(r, filename, dataIn, colorize, rawErrors)
	fmt.Fprintf(os.Stderr, "%s", errorString)

	if err != nil {
		fail("Error translating config: %v\n", err)
	}
	if strict && len(r.Entries) > 0 {
		fail("Config produced warnings and --strict was specified\n")
	}

	if !check {
		outfile := os.Stdout
		if output != "" {
			var err error
			outfile, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				fail("failed to open %s: %v\n", output, err)
			}
			defer outfile.Close()
		}

		if _, err := outfile.Write(append(dataOut, '\n')); err != nil {
			fail("Failed to write config to %s: %v\n", outfile.Name(), err)
		}
	}
}
