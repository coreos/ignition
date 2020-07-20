// Copyright 2020 Red Hat, Inc.
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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var (
	flagRoot string
)

func main() {
	flag.StringVar(&flagRoot, "root", "", "Apply changes in the CHROOT_DIR directory and use the configuration files from the CHROOT_DIR directory")

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Printf("incorrectly called\n")
		os.Exit(1)
	}

	groupname := flag.Args()[0]

	groupContents, err := ioutil.ReadFile(path.Join(flagRoot, "/etc/group"))
	if err != nil {
		fmt.Printf("couldn't open /etc/group: %v\n", err)
		os.Exit(1)
	}
	modifiedGroupContent, err := skipGroupName(groupContents, 4, groupname)
	if err != nil {
		os.Exit(1)
	}
	groupFile, err := os.OpenFile(path.Join(flagRoot, "/etc/group"), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("couldn't open group file: %v\n", err)
		os.Exit(1)
	}
	defer groupFile.Close()
	_, err = groupFile.Write([]byte(fmt.Sprintf("%s\n", strings.Join(modifiedGroupContent, "\n"))))
	if err != nil {
		fmt.Printf("couldn't write to group file: %v\n", err)
		os.Exit(1)
	}

	gshadowContents, err := ioutil.ReadFile(path.Join(flagRoot, "/etc/gshadow"))
	if err != nil {
		fmt.Printf("couldn't open /etc/gshadow: %v\n", err)
		os.Exit(1)
	}

	modifiedGShadowContent, err := skipGroupName(gshadowContents, 4, groupname)
	if err != nil {
		os.Exit(1)
	}

	gshadowFile, err := os.OpenFile(path.Join(flagRoot, "/etc/gshadow"), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("couldn't open gshadow file: %v\n", err)
		os.Exit(1)
	}
	defer gshadowFile.Close()
	_, err = gshadowFile.Write([]byte(fmt.Sprintf("%s\n", strings.Join(modifiedGShadowContent, "\n"))))
	if err != nil {
		fmt.Printf("couldn't write to gshadow file: %v\n", err)
		os.Exit(1)
	}
}

// skipGroupName will skip the groupname from `/etc/{group/gshadow}`file which
// needs to be deleted from the system.
func skipGroupName(content []byte, colon int, username string) ([]string, error) {
	var finalContents []string
	contents := strings.Split(string(content), "\n")
	for i, l := range contents {
		if i == len(contents)-1 {
			// The last line is empty
			break
		}
		tokens := strings.Split(l, ":")
		if len(tokens) != colon {
			return nil, fmt.Errorf("incorrect number of items: %d", len(tokens))
		}

		if tokens[0] == username {
			continue
		}

		finalContents = append(finalContents, l)
	}
	return finalContents, nil
}
