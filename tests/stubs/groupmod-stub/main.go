// Copyright 2025 CoreOS, Inc.
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
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	flagRoot     string
	flagPassword string
	flagGid      int
)

func main() {
	flag.StringVar(&flagRoot, "root", "/", "Apply changes in the CHROOT_DIR directory and use the configuration files from the CHROOT_DIR directory")
	flag.StringVar(&flagPassword, "password", "", "The encrypted password for the group")
	flag.IntVar(&flagGid, "gid", -1, "The numerical value of the group's ID")

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Printf("incorrectly called\n")
		os.Exit(1)
	}

	groupname := flag.Args()[0]

	// Modify /etc/group
	groupContents, err := os.ReadFile(path.Join(flagRoot, "/etc/group"))
	if err != nil {
		fmt.Printf("couldn't open /etc/group: %v\n", err)
		os.Exit(1)
	}
	groupLines := strings.Split(string(groupContents), "\n")
	for i, l := range groupLines {
		if i == len(groupLines)-1 {
			// The last line is empty
			break
		}
		tokens := strings.Split(l, ":")
		if len(tokens) != 4 {
			fmt.Printf("scanned incorrect number of items in group: %d\n", len(tokens))
			os.Exit(1)
		}
		currGroup := tokens[0]
		currGid := tokens[2]
		currMembers := tokens[3]

		if currGroup != groupname {
			continue
		}

		if flagGid != -1 {
			currGid = strconv.Itoa(flagGid)
		}

		newGroupLine := fmt.Sprintf("%s:x:%s:%s", currGroup, currGid, currMembers)

		groupLines[i] = newGroupLine
	}

	groupFile, err := os.OpenFile(path.Join(flagRoot, "/etc/group"), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("couldn't open group file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = groupFile.Close()
	}()

	_, err = groupFile.Write([]byte(strings.Join(groupLines, "\n")))
	if err != nil {
		fmt.Printf("couldn't write to group file: %v\n", err)
		os.Exit(1)
	}

	// Modify /etc/gshadow
	gshadowContents, err := os.ReadFile(path.Join(flagRoot, "/etc/gshadow"))
	if err != nil {
		fmt.Printf("couldn't open /etc/gshadow: %v\n", err)
		os.Exit(1)
	}
	gshadowLines := strings.Split(string(gshadowContents), "\n")
	for i, l := range gshadowLines {
		if i == len(gshadowLines)-1 {
			// The last line is empty
			break
		}
		tokens := strings.Split(l, ":")
		if len(tokens) != 4 {
			fmt.Printf("scanned incorrect number of items in gshadow: %d\n", len(tokens))
			os.Exit(1)
		}
		currGroup := tokens[0]
		currPassword := tokens[1]
		currAdmins := tokens[2]
		currMembers := tokens[3]

		if currGroup != groupname {
			continue
		}

		if flagPassword != "" {
			currPassword = flagPassword
		}

		newGshadowLine := fmt.Sprintf("%s:%s:%s:%s", currGroup, currPassword, currAdmins, currMembers)

		gshadowLines[i] = newGshadowLine
	}

	gshadowFile, err := os.OpenFile(path.Join(flagRoot, "/etc/gshadow"), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("couldn't open gshadow file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = gshadowFile.Close()
	}()

	_, err = gshadowFile.Write([]byte(strings.Join(gshadowLines, "\n")))
	if err != nil {
		fmt.Printf("couldn't write to gshadow file: %v\n", err)
		os.Exit(1)
	}
}
