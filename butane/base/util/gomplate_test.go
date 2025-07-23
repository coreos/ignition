package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func preserveGlobals(t *testing.T) func() {
	t.Helper()
	oldConfigPath := GomplateConfigPath
	oldRenderer := renderer
	oldContext := renderContext

	return func() {
		GomplateConfigPath = oldConfigPath
		renderer = oldRenderer
		renderContext = oldContext
	}
}

func initGomplate(t *testing.T, gomplateConfig string) error {
	t.Helper()
	EnableGomplate = true

	if gomplateConfig != "" {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".gomplate.yaml")

		err := os.WriteFile(configPath, []byte(gomplateConfig), 0644)
		if err != nil {
			return err
		}

		GomplateConfigPath = configPath
	} else {
		GomplateConfigPath = ""
	}

	return InitGomplateRenderer()
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

	output, err := GomplateReadLocalFile(tmpFile)
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
	EnableGomplate = false

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
	EnableGomplate = true
	GomplateConfigPath = "/nonexistent/path/.gomplate.yaml"

	err := InitGomplateRenderer()
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
