// Copyright 2014-2015 VMware, Inc. All Rights Reserved.
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

// Originally from https://github.com/vmware-archive/vmw-ovflib

package vmware

import (
	"encoding/xml"
	"fmt"

	"github.com/beevik/etree"
)

const (
	XMLNS = "http://schemas.dmtf.org/ovf/environment/1"
)

type environment struct {
	Platform   platform   `xml:"PlatformSection"`
	Properties []property `xml:"PropertySection>Property"`
}

type platform struct {
	Kind    string `xml:"Kind"`
	Version string `xml:"Version"`
	Vendor  string `xml:"Vendor"`
	Locale  string `xml:"Locale"`
}

type property struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

type OvfEnvironment struct {
	Platform   platform
	Properties map[string]string
}

func ReadOvfEnvironment(doc []byte) (OvfEnvironment, error) {
	var env environment
	if err := xml.Unmarshal(doc, &env); err != nil {
		return OvfEnvironment{}, err
	}

	dict := make(map[string]string)
	for _, p := range env.Properties {
		dict[p.Key] = p.Value
	}
	return OvfEnvironment{Properties: dict, Platform: env.Platform}, nil
}

// Return the new OVF document, and true if anything was deleted.
func DeleteOvfProperties(from []byte, props []string) ([]byte, bool, error) {
	// parse document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(from); err != nil {
		return nil, false, fmt.Errorf("parsing OVF environment: %w", err)
	}

	// build set of properties to drop
	drops := make(map[string]struct{})
	for _, prop := range props {
		drops[prop] = struct{}{}
	}

	// etree's XPath implementation isn't rigorous about keeping
	// separate namespaces separate.  Be extra-careful to check
	// namespace URIs everywhere.
	removed := false
	for _, parent := range doc.FindElements("/Environment[namespace-uri()='" + XMLNS + "']/PropertySection[namespace-uri()='" + XMLNS + "']") {
		// walk each property
		var remove []*etree.Element
		for _, el := range parent.ChildElements() {
			if el.NamespaceURI() != XMLNS {
				continue
			}
			// walk attrs by hand so we can check namespaces
			for _, attr := range el.Attr {
				if attr.NamespaceURI() != XMLNS {
					continue
				}
				// queue property for removal if it's on the
				// list
				if attr.Key == "key" {
					if _, drop := drops[attr.Value]; drop {
						remove = append(remove, el)
						removed = true
					}
					break
				}
			}
		}
		// remove queued properties
		for _, el := range remove {
			parent.RemoveChild(el)
		}
	}

	// Out of caution, if we didn't find anything to remove, return
	// the input bytes rather than reserializing.
	if !removed {
		return from, removed, nil
	}

	to, err := doc.WriteToBytes()
	if err != nil {
		return nil, false, fmt.Errorf("serializing OVF environment: %w", err)
	}
	return to, removed, nil
}
