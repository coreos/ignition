// Copyright 2015 CoreOS, Inc.
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

package systemd

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/unit"
)

// WaitOnDevices waits for the devices named in devs to be plugged before returning.
func WaitOnDevices(ctx context.Context, devs []string, stage string) error {
	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return err
	}

	results := map[string]chan string{}
	for _, dev := range devs {
		unitName := unit.UnitNamePathEscape(dev + ".device")
		results[unitName] = make(chan string, 1)

		if _, err = conn.StartUnitContext(ctx, unitName, "replace", results[unitName]); err != nil {
			return fmt.Errorf("failed starting device unit %s: %v", unitName, err)
		}
	}

	for unitName, result := range results {
		s := <-result

		if s != "done" {
			return fmt.Errorf("device unit %s %s", unitName, s)
		}
	}

	return nil
}

// GetSystemdVersion fetches the version of Systemd
// in a given system.
func GetSystemdVersion(ctx context.Context) (uint, error) {
	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return 0, err
	}
	version, err := conn.GetManagerProperty("Version")
	if err != nil {
		return 0, err
	}
	// Handle different systemd versioning schemes that are being returned.
	// for e.g:
	// - Fedora 31: `"v243.5-1.fc31"`
	// - RHEL 8: `"239"`
	re := regexp.MustCompile(`\d+`)
	systemdVersion := re.FindString(version)
	value, err := strconv.Atoi(systemdVersion)
	if err != nil {
		return 0, err
	}
	return uint(value), nil
}
