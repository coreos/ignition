// Copyright 2026 Red Hat, Inc
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

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ign36 "github.com/coreos/ignition/v2/config/v3_6"
	ign36types "github.com/coreos/ignition/v2/config/v3_6/types"
	"github.com/vincent-petithory/dataurl"
)

func TestFileProcessorCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the processor fixtures use POSIX shell scripts")
	}

	dir := t.TempDir()
	butane := filepath.Join(dir, "butane")
	build := exec.Command("go", "build", "-o", butane, ".")
	build.Dir = "."
	buildOutput, err := build.CombinedOutput()
	cliMustNoError(t, err, fmt.Sprintf("building Butane: %s", buildOutput))

	processor := filepath.Join(dir, "processor")
	cliWrite(t, processor, "#!/bin/sh\nsed 's/processor-placeholder/processor-rendered/g'\n", 0755)
	failing := filepath.Join(dir, "failing-processor")
	cliWrite(t, failing, "#!/bin/sh\ncat >/dev/null\necho processor-failed >&2\nexit 23\n", 0755)
	validInput := filepath.Join(dir, "valid.bu")
	cliWrite(t, validInput, cliConfigWithInline("ordinary config"), 0644)

	t.Run("processor and nonempty allowlist are required together", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
		}{
			{name: "processor without allowlist", args: []string{"--file-processor", processor}},
			{name: "allowlist without processor", args: []string{"--file-processor-allowed-ext=bu"}},
			{name: "empty allowlist", args: []string{"--file-processor", processor, "--file-processor-allowed-ext="}},
			{name: "empty processor", args: []string{"--file-processor=", "--file-processor-allowed-ext=bu"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, stderr, err := cliRun(butane, nil, append([]string{"--raw"}, append(tc.args, validInput)...)...)
				cliMustError(t, err)
				if len(stderr) == 0 {
					t.Fatal("expected a CLI validation error")
				}
			})
		}
	})

	t.Run("without processor flags local and binary tree bytes are unchanged", func(t *testing.T) {
		filesDir := filepath.Join(dir, "default-files")
		cliMustNoError(t, os.MkdirAll(filepath.Join(filesDir, "tree"), 0755), "creating default files")
		binaryData := bytes.Repeat([]byte{0x00, 0x01, 0xfe, 0xff, 0x0a, 0x0d}, 100)
		cliWrite(t, filepath.Join(filesDir, "payload.txt"), "processor-placeholder", 0644)
		cliMustNoError(t, os.WriteFile(filepath.Join(filesDir, "tree", "payload.bin"), binaryData, 0644), "writing binary tree file")
		input := filepath.Join(dir, "default.bu")
		cliWrite(t, input, `variant: fcos
version: 1.7.0
storage:
  files:
    - path: /etc/local
      contents:
        local: payload.txt
  trees:
    - local: tree
      path: /etc/tree
`, 0644)
		stdout, stderr, err := cliRun(butane, nil, "--raw", "--files-dir", filesDir, input)
		cliMustNoError(t, err, string(stderr))
		cfg := cliParseConfig(t, stdout)
		cliMustEqual(t, "processor-placeholder", string(cliContents(t, cfg, "/etc/local")))
		cliMustBytesEqual(t, binaryData, cliContents(t, cfg, "/etc/tree/payload.bin"))
	})

	t.Run("named root file suffix matching is multipart, optional-dot, and case-sensitive", func(t *testing.T) {
		cases := []struct {
			file string
			ext  string
			want bool
		}{
			{file: "root.templated.yml", ext: "templated.yml", want: true},
			{file: "root.templated.yml", ext: ".templated.yml", want: true},
			{file: "root.BU", ext: "bu", want: false},
		}
		for _, tc := range cases {
			t.Run(tc.file+"-"+tc.ext, func(t *testing.T) {
				input := filepath.Join(dir, tc.file)
				cliWrite(t, input, cliConfigWithInline("processor-placeholder"), 0644)
				stdout, stderr, err := cliRun(butane, nil, "--raw", "--file-processor", processor, "--file-processor-allowed-ext="+tc.ext, input)
				cliMustNoError(t, err, string(stderr))
				cfg := cliParseConfig(t, stdout)
				want := "processor-placeholder"
				if tc.want {
					want = "processor-rendered"
				}
				cliMustEqual(t, want, string(cliContents(t, cfg, "/etc/message")))
			})
		}
	})

	t.Run("processed root content is compressed only after processing", func(t *testing.T) {
		input := filepath.Join(dir, "compression.bu")
		payload := strings.Repeat("processor-placeholder ", 99) + "processor-placeholder"
		cliWrite(t, input, cliConfigWithInline(payload), 0644)
		stdout, stderr, err := cliRun(butane, nil, "--raw", "--file-processor", processor, "--file-processor-allowed-ext=bu", input)
		cliMustNoError(t, err, string(stderr))
		cfg := cliParseConfig(t, stdout)
		cliMustEqual(t, strings.ReplaceAll(payload, "processor-placeholder", "processor-rendered"), string(cliContents(t, cfg, "/etc/message")))
		if !cliIsGzip(t, cfg, "/etc/message") {
			t.Fatal("expected processed resource to be gzip-compressed")
		}
	})

	t.Run("wildcard selects named input", func(t *testing.T) {
		input := filepath.Join(dir, "wildcard.cfg")
		cliWrite(t, input, cliConfigWithInline("processor-placeholder"), 0644)
		stdout, stderr, err := cliRun(butane, nil, "--raw", "--file-processor", processor, "--file-processor-allowed-ext=*", input)
		cliMustNoError(t, err, string(stderr))
		cliMustEqual(t, "processor-rendered", string(cliContents(t, cliParseConfig(t, stdout), "/etc/message")))
	})

	t.Run("stdin is processed only by the explicit stdin sentinel", func(t *testing.T) {
		input := []byte(cliConfigWithInline("processor-placeholder"))
		for _, tc := range []struct {
			name string
			ext  string
			want string
		}{
			{name: "named suffix does not match stdin", ext: "bu", want: "processor-placeholder"},
			{name: "wildcard is for named files", ext: "*", want: "processor-placeholder"},
			{name: "stdin sentinel", ext: "stdin", want: "processor-rendered"},
			{name: "wildcard plus stdin sentinel", ext: "*,stdin", want: "processor-rendered"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				stdout, stderr, err := cliRun(butane, input, "--raw", "--file-processor", processor, "--file-processor-allowed-ext="+tc.ext)
				cliMustNoError(t, err, string(stderr))
				cliMustEqual(t, tc.want, string(cliContents(t, cliParseConfig(t, stdout), "/etc/message")))
			})
		}
	})

	t.Run("selected local files are processed and unlisted local files pass through", func(t *testing.T) {
		filesDir := filepath.Join(dir, "local-files")
		cliMustNoError(t, os.MkdirAll(filesDir, 0755), "creating local files")
		cliWrite(t, filepath.Join(filesDir, "selected.txt"), "processor-placeholder", 0644)
		cliWrite(t, filepath.Join(filesDir, "unlisted.raw"), "processor-placeholder", 0644)
		input := filepath.Join(dir, "local.bu")
		cliWrite(t, input, `variant: fcos
version: 1.7.0
storage:
  files:
    - path: /etc/selected
      contents:
        local: selected.txt
    - path: /etc/unlisted
      contents:
        local: unlisted.raw
`, 0644)
		stdout, stderr, err := cliRun(butane, nil, "--raw", "--files-dir", filesDir, "--file-processor", processor, "--file-processor-allowed-ext=bu,txt", input)
		cliMustNoError(t, err, string(stderr))
		cfg := cliParseConfig(t, stdout)
		cliMustEqual(t, "processor-rendered", string(cliContents(t, cfg, "/etc/selected")))
		cliMustEqual(t, "processor-placeholder", string(cliContents(t, cfg, "/etc/unlisted")))
	})

	t.Run("selected tree files are processed while binary files pass through byte-for-byte", func(t *testing.T) {
		filesDir := filepath.Join(dir, "tree-files")
		treeDir := filepath.Join(filesDir, "tree")
		cliMustNoError(t, os.MkdirAll(filepath.Join(treeDir, "nested", "deeper"), 0755), "creating nested tree")
		binaryData := bytes.Repeat([]byte{0x00, 0x01, 0xfe, 0xff, 0x0a, 0x0d}, 100)
		cliWrite(t, filepath.Join(treeDir, "selected.txt"), "processor-placeholder", 0644)
		cliMustNoError(t, os.WriteFile(filepath.Join(treeDir, "binary.bin"), binaryData, 0644), "writing binary tree file")
		cliWrite(t, filepath.Join(treeDir, "nested", "selected.txt"), "processor-placeholder nested", 0644)
		cliWrite(t, filepath.Join(treeDir, "nested", "unlisted.raw"), "processor-placeholder keep", 0644)
		cliMustNoError(t, os.WriteFile(filepath.Join(treeDir, "nested", "deeper", "binary.bin"), binaryData, 0644), "writing nested binary tree file")
		input := filepath.Join(dir, "tree.bu")
		cliWrite(t, input, "variant: fcos\nversion: 1.7.0\nstorage:\n  trees:\n    - local: tree\n      path: /etc/tree\n", 0644)
		stdout, stderr, err := cliRun(butane, nil, "--raw", "--files-dir", filesDir, "--file-processor", processor, "--file-processor-allowed-ext=bu,txt", input)
		cliMustNoError(t, err, string(stderr))
		cfg := cliParseConfig(t, stdout)
		cliMustEqual(t, "processor-rendered", string(cliContents(t, cfg, "/etc/tree/selected.txt")))
		cliMustBytesEqual(t, binaryData, cliContents(t, cfg, "/etc/tree/binary.bin"))
		cliMustEqual(t, "processor-rendered nested", string(cliContents(t, cfg, "/etc/tree/nested/selected.txt")))
		cliMustEqual(t, "processor-placeholder keep", string(cliContents(t, cfg, "/etc/tree/nested/unlisted.raw")))
		cliMustBytesEqual(t, binaryData, cliContents(t, cfg, "/etc/tree/nested/deeper/binary.bin"))
	})

	t.Run("root, local, and tree processor failures include source filename", func(t *testing.T) {
		rootInput := filepath.Join(dir, "root-failure.bu")
		cliWrite(t, rootInput, cliConfigWithInline("ordinary config"), 0644)
		_, stderr, err := cliRun(butane, nil, "--raw", "--file-processor", failing, "--file-processor-allowed-ext=bu", rootInput)
		cliMustError(t, err)
		cliMustContain(t, string(stderr), "processor-failed")
		cliMustContain(t, string(stderr), filepath.Base(rootInput))

		filesDir := filepath.Join(dir, "failure-files")
		treeDir := filepath.Join(filesDir, "tree")
		cliMustNoError(t, os.MkdirAll(treeDir, 0755), "creating failing local files")
		cliWrite(t, filepath.Join(filesDir, "local-error.txt"), "local failure", 0644)
		cliWrite(t, filepath.Join(treeDir, "tree-error.txt"), "tree failure", 0644)
		localInput := filepath.Join(dir, "local-failure.bu")
		cliWrite(t, localInput, "variant: fcos\nversion: 1.7.0\nstorage:\n  files:\n    - path: /etc/local\n      contents:\n        local: local-error.txt\n", 0644)
		_, stderr, err = cliRun(butane, nil, "--raw", "--files-dir", filesDir, "--file-processor", failing, "--file-processor-allowed-ext=txt", localInput)
		cliMustError(t, err)
		cliMustContain(t, string(stderr), "processor-failed")
		cliMustContain(t, string(stderr), "local-error.txt")

		treeInput := filepath.Join(dir, "tree-failure.bu")
		cliWrite(t, treeInput, "variant: fcos\nversion: 1.7.0\nstorage:\n  trees:\n    - local: tree\n      path: /etc/tree\n", 0644)
		_, stderr, err = cliRun(butane, nil, "--raw", "--files-dir", filesDir, "--file-processor", failing, "--file-processor-allowed-ext=txt", treeInput)
		cliMustError(t, err)
		cliMustContain(t, string(stderr), "processor-failed")
		cliMustContain(t, string(stderr), "tree-error.txt")
	})

	t.Run("missing processor executable is reported", func(t *testing.T) {
		input := filepath.Join(dir, "missing-executable.bu")
		cliWrite(t, input, cliConfigWithInline("ordinary config"), 0644)
		missing := filepath.Join(dir, "missing-processor")
		_, stderr, err := cliRun(butane, nil, "--raw", "--file-processor", missing, "--file-processor-allowed-ext=bu", input)
		cliMustError(t, err)
		cliMustContain(t, string(stderr), "file processor")
		cliMustContain(t, string(stderr), filepath.Base(input))
	})
}

func cliRun(binary string, stdin []byte, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func cliConfigWithInline(value string) string {
	return fmt.Sprintf("variant: fcos\nversion: 1.7.0\nstorage:\n  files:\n    - path: /etc/message\n      contents:\n        inline: %q\n", value)
}

func cliParseConfig(t *testing.T, raw []byte) ign36types.Config {
	t.Helper()
	cfg, _, err := ign36.Parse(raw)
	cliMustNoError(t, err, string(raw))
	return cfg
}

func cliContents(t *testing.T, cfg ign36types.Config, path string) []byte {
	t.Helper()
	for _, file := range cfg.Storage.Files {
		if file.Path != path || file.Contents.Source == nil {
			continue
		}
		data, err := dataurl.DecodeString(*file.Contents.Source)
		cliMustNoError(t, err, "decoding data URL")
		if file.Contents.Compression == nil || *file.Contents.Compression != "gzip" {
			return data.Data
		}
		reader, err := gzip.NewReader(bytes.NewReader(data.Data))
		cliMustNoError(t, err, "opening gzip resource")
		contents, err := io.ReadAll(reader)
		cliMustNoError(t, err, "reading gzip resource")
		cliMustNoError(t, reader.Close(), "closing gzip resource")
		return contents
	}
	t.Fatalf("Ignition file %q not found", path)
	return nil
}

func cliIsGzip(t *testing.T, cfg ign36types.Config, path string) bool {
	t.Helper()
	for _, file := range cfg.Storage.Files {
		if file.Path == path {
			return file.Contents.Compression != nil && *file.Contents.Compression == "gzip"
		}
	}
	t.Fatalf("Ignition file %q not found", path)
	return false
}

func cliWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	cliMustNoError(t, os.MkdirAll(filepath.Dir(path), 0755), "creating parent directory")
	cliMustNoError(t, os.WriteFile(path, []byte(content), mode), "writing test fixture")
}

func cliMustNoError(t testing.TB, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", context, err)
	}
}

func cliMustError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected command to fail")
	}
}

func cliMustContain(t testing.TB, value, part string) {
	t.Helper()
	if !strings.Contains(value, part) {
		t.Fatalf("expected %q to contain %q", value, part)
	}
}

func cliMustEqual(t testing.TB, want, got string) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func cliMustBytesEqual(t testing.TB, want, got []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("byte content differs: got %d bytes, want %d", len(got), len(want))
	}
}
