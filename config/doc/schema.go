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

	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/go-semver/semver"
	"github.com/mitchellh/copystructure"
	"gopkg.in/yaml.v3"
)

//go:embed ignition.yaml
var ignitionDocs []byte

const IGNITION_VARIANT = "ignition"
const ROOT_COMPONENT = "root"

type Components map[string]DocNode

type DocNode struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"desc"`
	Required    *bool       `yaml:"required"`
	RequiredIf  Constraints `yaml:"required-if"`
	Skip        Constraints `yaml:"unless"`
	Transforms  []Transform `yaml:"transforms"`
	Children    []DocNode   `yaml:"children"`

	Component string `yaml:"use"`
	After     string `yaml:"after"`

	// populated after component resolution
	Parent *DocNode
}

type Transform struct {
	Regex       string      `yaml:"regex"`
	Replacement string      `yaml:"replacement"`
	Constraints Constraints `yaml:"if"`
	Descendants bool        `yaml:"descendants"`
}

type Constraints []Constraint

type Constraint struct {
	Variant string  `yaml:"variant"`
	MinVer  *string `yaml:"min"`
	MaxVer  *string `yaml:"max"`
}

type VariantVersions map[string]semver.Version

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

func (comps Components) Generate(vers VariantVersions, config any, w io.Writer) error {
	root, err := comps.resolve()
	if err != nil {
		return err
	}
	gen := generator{
		vers: vers,
		w:    w,
	}
	return gen.descendNode(root, reflect.TypeOf(config), nil)
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
	// now that all merging has happened, any remaining After field is
	// incorrect
	if node.After != "" {
		return fmt.Errorf("field %q: stray `after` parameter %q", node.Name, node.After)
	}
	// descend children
	for i := range node.Children {
		if err := comps.resolveComponents(&node.Children[i]); err != nil {
			return err
		}
	}
	return nil
}

func (comps Components) Merge(override Components) error {
	for name, comp := range comps {
		overrideComp, ok := override[name]
		if !ok {
			// no override
			continue
		}
		// present in both
		if err := comp.merge(overrideComp); err != nil {
			return fmt.Errorf("merging component %q: %w", name, err)
		}
		comps[name] = comp
	}
	for name, comp := range override {
		if _, ok := comps[name]; ok {
			// present in both
			continue
		}
		// only present in override; add to current
		comps[name] = comp
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

func (node *DocNode) renderDescription(vers VariantVersions) (string, error) {
	desc := node.Description
	for _, xfrm := range node.transforms() {
		matches, err := xfrm.Constraints.matches(vers)
		if err != nil {
			return "", fmt.Errorf("field %q: %w", node.Name, err)
		}
		if util.IsFalse(matches) {
			continue
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
	// substitute version variables last, so transforms can add them
	for variant, ver := range vers {
		desc = strings.ReplaceAll(desc, fmt.Sprintf("%%%s_version%%", variant), ver.String())
	}
	return desc, nil
}

func (node *DocNode) required(vers VariantVersions) (*bool, error) {
	conditional, err := node.RequiredIf.matches(vers)
	if err != nil {
		return nil, err
	}
	if conditional != nil {
		return conditional, nil
	}
	return node.Required, nil
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
		node.RequiredIf = nil
	}
	if len(override.RequiredIf) > 0 {
		node.Required = nil
		node.RequiredIf = append(node.RequiredIf, override.RequiredIf...)
	}
	node.Skip = append(node.Skip, override.Skip...)
	node.Transforms = append(node.Transforms, override.Transforms...)
	if override.Component != "" {
		node.Component = override.Component
	}
	if override.After != "" {
		node.After = override.After
	}

	// insertions and overrides for children
	unseenOverrides := make(map[string]DocNode)
	overrideByName := make(map[string]DocNode)
	insertionsByPredecessor := make(map[string][]DocNode)
	for _, child := range override.Children {
		unseenOverrides[child.Name] = child
		overrideByName[child.Name] = child
		if child.After != "" {
			insertionsByPredecessor[child.After] = append(insertionsByPredecessor[child.After], child)
		}
	}
	var children []DocNode
	insert := func(predecessor string) {
		for _, child := range insertionsByPredecessor[predecessor] {
			child.After = ""
			children = append(children, child)
			delete(unseenOverrides, child.Name)
		}
	}
	insert("^")
	for _, child := range node.Children {
		if override, ok := overrideByName[child.Name]; ok {
			if override.After != "" {
				return fmt.Errorf("field %q: override %q sets `after` and also matches existing field", node.Name, child.Name)
			}
			if err := child.merge(override); err != nil {
				return err
			}
			delete(unseenOverrides, child.Name)
		}
		children = append(children, child)
		insert(child.Name)
	}
	insert("$")
	node.Children = children

	// find unused overrides
	for _, child := range unseenOverrides {
		if child.After != "" {
			return fmt.Errorf("field %q: child %q: `after` field %q not found", node.Name, child.Name, child.After)
		} else {
			return fmt.Errorf("field %q: override %q not found; did you mean to set `after`?", node.Name, child.Name)
		}
	}

	return nil
}

func (cons Constraints) matches(vers VariantVersions) (*bool, error) {
	if len(cons) == 0 {
		// no constraints
		return nil, nil
	}
	for _, con := range cons {
		if con.Variant == "" {
			return nil, fmt.Errorf("missing `variant` in constraint")
		}
		ver, ok := vers[con.Variant]
		if !ok {
			// constraint isn't relevant to us
			continue
		}
		if con.MinVer != nil {
			min, err := semver.NewVersion(*con.MinVer)
			if err != nil {
				return nil, fmt.Errorf("parsing min %q: %w", *con.MinVer, err)
			}
			if ver.LessThan(*min) {
				// constraint failed; try others
				continue
			}
		}
		if con.MaxVer != nil {
			max, err := semver.NewVersion(*con.MaxVer)
			if err != nil {
				return nil, fmt.Errorf("parsing max %q: %w", *con.MaxVer, err)
			}
			if max.LessThan(ver) {
				// constraint failed; try others
				continue
			}
		}
		// one constraint matched; accept
		return util.BoolToPtr(true), nil
	}
	// no constraints matched; reject
	return util.BoolToPtr(false), nil
}
