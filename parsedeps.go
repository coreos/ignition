// Helper for parsing glide.lock file and spitting out
// bundled provides statements for an rpm spec file.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"

	yaml "gopkg.in/yaml.v2"
)

type Import struct {
	Name        string
	Version     string
	Subpackages []string
}

type Glide struct {
	Hash        string
	Updated     string
	Imports     []Import
	TestImports []Import
}

func main() {
	yamlFile, err := ioutil.ReadFile("glide.lock")
	if err != nil {
		log.Fatal(err)
	}

	var glide Glide
	err = yaml.Unmarshal(yamlFile, &glide)
	if err != nil {
		log.Fatal(err)
	}

	for _, imp := range glide.Imports {
		// we need format like this:
		// Provides: bundled(golang(github.com/coreos/go-oidc/oauth2)) = %{version}-5cf2aa52da8c574d3aa4458f471ad6ae2240fe6b
		for _, subp := range imp.Subpackages {
			name := path.Join(imp.Name, subp)
			fmt.Printf("Provides: bundled(golang(%s)) = %s-%s\n", name, "%{version}", imp.Version)
		}
	}
}
