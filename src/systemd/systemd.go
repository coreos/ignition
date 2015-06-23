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
	"fmt"

	"github.com/coreos/ignition/third_party/github.com/coreos/go-systemd/dbus"
	"github.com/coreos/ignition/third_party/github.com/coreos/go-systemd/unit"
)

// WaitOnDevices waits for the devices named in devs to be plugged before returning.
func WaitOnDevices(devs []string, stage string) error {
	conn, err := dbus.NewSystemdConnection()
	if err != nil {
		return err
	}

	devUnits := []string{}
	for _, d := range devs {
		devUnits = append(devUnits, unit.UnitNamePathEscape(d)+".device")
	}

	unitName := unit.UnitNameEscape(fmt.Sprintf("ignition_%s.service", stage))
	props := []dbus.Property{
		dbus.PropExecStart([]string{"/bin/true"}, false), // XXX(vc): we apparently are required to ExecStart _something_
		dbus.PropAfter(devUnits...),
		dbus.PropRequires(devUnits...),
	}

	res := make(chan string)
	if _, err = conn.StartTransientUnit(unitName, "replace", props, res); err != nil {
		return fmt.Errorf("failed creating transient unit %s: %v", unitName, err)
	}
	s := <-res

	if s != "done" {
		return fmt.Errorf("transient unit %s %s", unitName, s)
	}

	return nil
}
