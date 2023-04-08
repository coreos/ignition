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

package generate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/coreos/go-semver/semver"
)

const ROOT_COMPONENT = "root"

type Components map[string]DocNode

type DocNode struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"desc"`
	Required    *bool       `yaml:"required"`
	Transforms  []Transform `yaml:"transforms"`
	Children    []DocNode   `yaml:"children"`
}

type Transform struct {
	Regex       string  `yaml:"regex"`
	Replacement string  `yaml:"replacement"`
	MinVer      *string `yaml:"min"`
	MaxVer      *string `yaml:"max"`
}

func (comps Components) Resolve() (DocNode, error) {
	root, ok := comps[ROOT_COMPONENT]
	if !ok {
		return DocNode{}, fmt.Errorf("missing component %q", ROOT_COMPONENT)
	}
	return root, nil
}

func (node *DocNode) RenderDescription(ver *semver.Version) (string, error) {
	desc := strings.ReplaceAll(node.Description, "%VERSION%", ver.String())
	for _, xfrm := range node.Transforms {
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
		if new == desc {
			return "", fmt.Errorf("field %q: applying %q: transform didn't change anything", node.Name, xfrm.Regex)
		}
		desc = new
	}
	return desc, nil
}
