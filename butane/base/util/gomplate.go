package util

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"github.com/hairyhenderson/gomplate/v5"
)

var (
	EnableGomplate = false

	GomplateConfigPath = ".gomplate.yaml"
	renderer           = gomplate.NewRenderer(gomplate.RenderOptions{})
	renderContext      = context.Background()
)

func parseGomplateConfig() (*gomplate.Config, error) {
	f, err := os.Open(GomplateConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	return gomplate.Parse(f)
}

func InitGomplateRenderer() error {
	config, err := parseGomplateConfig()
	if err != nil {
		renderer = nil
		return err
	}

	if config != nil {
		if config.Experimental {
			renderContext = gomplate.SetExperimental(renderContext)
		}

		// Inspired by `gomplate.bindPlugins`
		funcMap := map[string]any{}
		for pluginName, plugin := range config.Plugins {
			// default the timeout to the one in the config
			timeout := config.PluginTimeout
			if plugin.Timeout != 0 {
				timeout = plugin.Timeout
			}

			funcMap[pluginName] = gomplate.PluginFunc(renderContext, plugin.Cmd, gomplate.PluginOpts{
				Timeout: timeout,
				Pipe:    plugin.Pipe,
				Stderr:  config.Stderr,
				Args:    plugin.Args,
			})
		}

		renderer = gomplate.NewRenderer(gomplate.RenderOptions{
			Funcs:       funcMap,
			Datasources: config.DataSources,
			Context:     config.Context,
			Templates:   config.Templates,
			LDelim:      config.LDelim,
			RDelim:      config.RDelim,
			MissingKey:  config.MissingKey,
		})
	}
	return nil
}

func GomplateReadLocalFile(file *os.File) ([]byte, error) {
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if !EnableGomplate {
		return fileContent, nil
	}

	var buf bytes.Buffer
	err = renderer.Render(renderContext, file.Name(), string(fileContent), &buf)
	return buf.Bytes(), err
}
