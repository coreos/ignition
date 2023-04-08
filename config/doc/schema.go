// Copyright 2023 Red Hat, Inc.
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

package doc

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/mitchellh/copystructure"
	"gopkg.in/yaml.v3"
)

//go:embed ignition.yaml
var ignitionDocs []byte

const ROOT_COMPONENT = "root"

type Components map[string]DocNode

type DocNode struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"desc"`
	Required    *bool       `yaml:"required"`
	Transforms  []Transform `yaml:"transforms"`
	Children    []DocNode   `yaml:"children"`

	Component string `yaml:"use"`

	// populated after component resolution
	Parent *DocNode
}

type Transform struct {
	Regex       string  `yaml:"regex"`
	Replacement string  `yaml:"replacement"`
	MinVer      *string `yaml:"min"`
	MaxVer      *string `yaml:"max"`
	Descendants bool    `yaml:"descendants"`
}

func IgnitionComponents() (Components, error) {
	return ParseComponents(bytes.NewBuffer(ignitionDocs))
}

func ParseComponents(r io.Reader) (Components, error) {
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	var comps Components
	if err := decoder.Decode(&comps); err != nil {
		return comps, fmt.Errorf("parsing components: %w", err)
	}
	return comps, nil
}

func (comps Components) Generate(ver *semver.Version, config any, w io.Writer) error {
	root, err := comps.resolve()
	if err != nil {
		return err
	}
	return descendNode(ver, root, reflect.TypeOf(config), 0, w)
}

func (comps Components) resolve() (DocNode, error) {
	root, ok := comps[ROOT_COMPONENT]
	if !ok {
		return DocNode{}, fmt.Errorf("missing component %q", ROOT_COMPONENT)
	}
	root = copystructure.Must(copystructure.Copy(root)).(DocNode)
	if err := comps.resolveComponents(&root); err != nil {
		return DocNode{}, err
	}
	root.setParentLinks()
	return root, nil
}

func (comps Components) resolveComponents(node *DocNode) error {
	// recursively insert the subtree of any component reference
	if node.Component != "" {
		comp, ok := comps[node.Component]
		if !ok {
			return fmt.Errorf("field %q: no such component %q", node.Name, node.Component)
		}
		if comp.Component != "" {
			return fmt.Errorf("component %q cannot itself refer to a component", node.Component)
		}
		comp = copystructure.Must(copystructure.Copy(comp)).(DocNode)
		if err := comp.merge(*node); err != nil {
			return err
		}
		comp.Component = ""
		*node = comp
	}
	// descend children
	for i := range node.Children {
		if err := comps.resolveComponents(&node.Children[i]); err != nil {
			return err
		}
	}
	return nil
}

func (node *DocNode) setParentLinks() {
	for i := range node.Children {
		child := &node.Children[i]
		child.Parent = node
		child.setParentLinks()
	}
}

func (node *DocNode) renderDescription(ver *semver.Version) (string, error) {
	desc := strings.ReplaceAll(node.Description, "%VERSION%", ver.String())
	for _, xfrm := range node.transforms() {
		if xfrm.MinVer != nil {
			min, err := semver.NewVersion(*xfrm.MinVer)
			if err != nil {
				return "", fmt.Errorf("field %q: parsing min %q: %w", node.Name, *xfrm.MinVer, err)
			}
			if ver.LessThan(*min) {
				continue
			}
		}
		if xfrm.MaxVer != nil {
			max, err := semver.NewVersion(*xfrm.MaxVer)
			if err != nil {
				return "", fmt.Errorf("field %q: parsing max %q: %w", node.Name, *xfrm.MaxVer, err)
			}
			if max.LessThan(*ver) {
				continue
			}
		}
		re, err := regexp.Compile(xfrm.Regex)
		if err != nil {
			return "", fmt.Errorf("field %q: compiling %q: %w", node.Name, xfrm.Regex, err)
		}
		new := re.ReplaceAllString(desc, xfrm.Replacement)
		if !xfrm.Descendants && new == desc {
			return "", fmt.Errorf("field %q: applying %q: transform didn't change anything", node.Name, xfrm.Regex)
		}
		desc = new
	}
	return desc, nil
}

func (node *DocNode) transforms() []Transform {
	var ret []Transform
	var descend func(node *DocNode, inheritedOnly bool)
	descend = func(node *DocNode, inheritedOnly bool) {
		for _, xfrm := range node.Transforms {
			if inheritedOnly && !xfrm.Descendants {
				continue
			}
			ret = append(ret, xfrm)
		}
		if node.Parent != nil {
			descend(node.Parent, true)
		}
	}
	descend(node, false)
	return ret
}

func (node *DocNode) merge(override DocNode) error {
	// merge fields
	if override.Name != "" {
		node.Name = override.Name
	}
	if override.Description != "" {
		node.Description = override.Description
	}
	if override.Required != nil {
		node.Required = override.Required
	}
	node.Transforms = append(node.Transforms, override.Transforms...)
	if override.Component != "" {
		node.Component = override.Component
	}

	// merge overrides for children
	overrideChildren := make(map[string]DocNode)
	for _, child := range override.Children {
		overrideChildren[child.Name] = child
	}
	for i := range node.Children {
		child := &node.Children[i]
		if override, ok := overrideChildren[child.Name]; ok {
			if err := child.merge(override); err != nil {
				return err
			}
			delete(overrideChildren, child.Name)
		}
	}

	// find unused overrides
	for _, child := range overrideChildren {
		return fmt.Errorf("field %q: override %q not found in component", node.Name, child.Name)
	}

	return nil
}
