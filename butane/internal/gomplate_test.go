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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	baseutil "github.com/coreos/ignition/v2/butane/base/util"
)

func preserveGlobals(t *testing.T) func() {
	t.Helper()
	oldEnableGomplate := enableGomplate
	oldConfigPath := gomplateConfigPath
	oldRenderer := renderer
	oldContext := renderContext

	return func() {
		enableGomplate = oldEnableGomplate
		gomplateConfigPath = oldConfigPath
		renderer = oldRenderer
		renderContext = oldContext
	}
}

func initGomplate(t *testing.T, gomplateConfig string) error {
	t.Helper()
	enableGomplate = true

	if gomplateConfig != "" {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".gomplate.yaml")

		err := os.WriteFile(configPath, []byte(gomplateConfig), 0644)
		if err != nil {
			return err
		}

		gomplateConfigPath = configPath
	} else {
		gomplateConfigPath = ""
	}

	return initGomplateRenderer()
}

func evalTemplate(t *testing.T, template string) (string, error) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "template-*.tmpl")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(template); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("failed to return to begining of temp file: %v", err)
	}

	output, err := gomplateReadFile(tmpFile)
	return string(output), err
}

func TestInvalidGomplateConfig(t *testing.T) {
	defer preserveGlobals(t)()
	if err := initGomplate(t, "not: valid: config: ["); err == nil {
		t.Fatalf("gomplate initialization should have failed: %v", err)
	}
}

func TestInvalidTemplate(t *testing.T) {
	defer preserveGlobals(t)()
	if err := initGomplate(t, "#empty config"); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	_, err := evalTemplate(t, "{{ .NonExistentField }}")

	if err == nil {
		t.Fatalf("expected error for missing key, got nil")
	}
}

func TestNoCustomConfig(t *testing.T) {
	defer preserveGlobals(t)()
	if err := initGomplate(t, "#empty config"); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	rendered, err := evalTemplate(t, `{{ "foobarbazquxquux" | strings.Abbrev 9 }}`)

	if err != nil {
		t.Fatalf("unexpected error: %+v\n", err)
	}

	if rendered != "foobar..." {
		t.Fatalf("Invalid rendered template, got: '%s'\n", rendered)
	}
}

func TestGomplateLocalFile(t *testing.T) {
	defer preserveGlobals(t)()
	baseutil.SetLocalFileReader(gomplateReadLocalFile)
	defer baseutil.SetLocalFileReader(nil)

	if err := initGomplate(t, "#empty config"); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "template"), []byte(`{{ "foo" | strings.ToUpper }}`), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}
	rendered, err := baseutil.ReadLocalFile("template", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(rendered) != "FOO" {
		t.Fatalf("Invalid rendered template, got: '%s'\n", rendered)
	}
}

func TestGomplateConfigApplication(t *testing.T) {
	defer preserveGlobals(t)()
	// Create a mock HTTP server that returns a fixed JSON response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"hello": "Hello"}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("json encoding failed: %v", err)
		}
	}))
	defer ts.Close()

	configContent := `
      leftDelim: ($(
      rightDelim: )$)
      context:
        data:
          url: ` + ts.URL
	if err := initGomplate(t, configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	rendered, err := evalTemplate(t, "($( .data.hello )$)!")

	if err != nil {
		t.Fatalf("unexpected error: %+v\n", err)
	}
	if rendered != "Hello!" {
		t.Fatalf("Invalid rendered template, got: '%s'\n", rendered)
	}
}

func TestGomplateDisabled(t *testing.T) {
	defer preserveGlobals(t)()
	enableGomplate = false

	expected := "some raw content"
	rendered, err := evalTemplate(t, expected)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rendered != expected {
		t.Fatalf("Invalid rendered template, got: '%s'\n", rendered)
	}
}

func TestMissingGomplateConfigFile(t *testing.T) {
	defer preserveGlobals(t)()
	enableGomplate = true
	gomplateConfigPath = "/nonexistent/path/.gomplate.yaml"

	err := initGomplateRenderer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGomplatePlugins(t *testing.T) {
	defer preserveGlobals(t)()

	configContent := `
      plugins:
        echo:
          cmd: /bin/echo
          args:
            - foo
    `
	if err := initGomplate(t, configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	// we also ensure no built-in functions got erased
	rendered, err := evalTemplate(t, `{{ echo "bar" | strings.Trunc 6 }}`)

	if err != nil {
		t.Fatalf("unexpected error: %+v\n", err)
	}
	if rendered != "foo ba" {
		t.Fatalf("Invalid rendered template, got: '%s'\n", rendered)
	}
}
