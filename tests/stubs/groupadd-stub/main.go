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
	flagSystem   bool
	flagPassword string
	flagGid      int
)

func main() {
	flag.StringVar(&flagRoot, "root", "", "Apply changes in the CHROOT_DIR directory and use the configuration files from the CHROOT_DIR directory")
	flag.BoolVar(&flagSystem, "system", false, "Create a system group")
	flag.StringVar(&flagPassword, "password", "", "The encrypted password for the group")
	flag.IntVar(&flagGid, "gid", -1, "The numerical value of the group's ID")

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Printf("incorrectly called\n")
		os.Exit(1)
	}

	groupname := flag.Args()[0]

	if flagGid == -1 {
		var err error
		flagGid, err = getNextGid()
		if err != nil {
			fmt.Printf("error getting next gid: %v\n", err)
			os.Exit(1)
		}
	}

	groupLine := fmt.Sprintf("%s:x:%d:\n", groupname, flagGid)

	groupFile, err := os.OpenFile(path.Join(flagRoot, "/etc/group"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("couldn't open group file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = groupFile.Close()
	}()
	_, err = groupFile.Write([]byte(groupLine))
	if err != nil {
		fmt.Printf("couldn't write to group file: %v\n", err)
		os.Exit(1)
	}

	gshadowPassword := "!"
	if flagPassword != "" {
		gshadowPassword = flagPassword
	}
	gshadowLine := fmt.Sprintf("%s:%s::\n", groupname, gshadowPassword)

	gshadowFile, err := os.OpenFile(path.Join(flagRoot, "/etc/gshadow"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("couldn't open gshadow file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = gshadowFile.Close()
	}()
	_, err = gshadowFile.Write([]byte(gshadowLine))
	if err != nil {
		fmt.Printf("couldn't write to gshadow file: %v\n", err)
		os.Exit(1)
	}
}

// getNextGid finds the next available gid starting at 1000 (or 201 for system groups)
func getNextGid() (int, error) {
	gidMap := make(map[int]struct{})

	groupContents, err := os.ReadFile(path.Join(flagRoot, "/etc/group"))
	if err != nil {
		return -1, err
	}
	groupLines := strings.Split(string(groupContents), "\n")
	for i, l := range groupLines {
		if i == len(groupLines)-1 {
			// the last line is empty
			break
		}
		// Will panic due to out of bounds if /etc/group is malformed
		tokens := strings.Split(l, ":")
		gid, err := strconv.ParseInt(tokens[2], 10, 64)
		if err != nil {
			return -1, err
		}
		gidMap[int(gid)] = struct{}{}
	}

	startGid := 1000
	if flagSystem {
		startGid = 201
	}

	for i := startGid; i < 65534; i++ {
		_, ok := gidMap[i]
		if !ok {
			return i, nil
		}
	}
	return -1, fmt.Errorf("out of gids")
}
