// Copyright 2020 CoreOS, Inc.
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

package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/coreos/ignition/v2/config"
	"github.com/coreos/ignition/v2/config/shared/errors"
	cfgutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
)

func TestParseInstanceUnit(t *testing.T) {
	type in struct {
		unit types.Unit
	}
	type out struct {
		unitName string
		instance string
		parseErr error
	}
	tests := []struct {
		in  in
		out out
	}{
		{in: in{types.Unit{Name: "echo@bar.service"}},
			out: out{unitName: "echo@.service", instance: "bar",
				parseErr: nil},
		},

		{in: in{types.Unit{Name: "echo@foo.service"}},
			out: out{unitName: "echo@.service", instance: "foo",
				parseErr: nil},
		},
		{in: in{types.Unit{Name: "echo.service"}},
			out: out{unitName: "", instance: "",
				parseErr: errors.ErrInvalidInstantiatedUnit},
		},
		{in: in{types.Unit{Name: "echo@fooservice"}},
			out: out{unitName: "", instance: "",
				parseErr: errors.ErrNoSystemdExt},
		},
		{in: in{types.Unit{Name: "echo@.service"}},
			out: out{unitName: "echo@.service", instance: "",
				parseErr: nil},
		},
		{in: in{types.Unit{Name: "postgresql@9.3-main.service"}},
			out: out{unitName: "postgresql@.service", instance: "9.3-main",
				parseErr: nil},
		},
	}
	for i, test := range tests {
		unitName, instance, err := parseInstanceUnit(test.in.unit)
		if test.out.parseErr != err {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out.parseErr, err)
		}
		if !reflect.DeepEqual(test.out.unitName, unitName) {
			t.Errorf("#%d: bad unitName: want %v, got %v", i, test.out.unitName, unitName)
		}
		if !reflect.DeepEqual(test.out.instance, instance) {
			t.Errorf("#%d: bad instance: want %v, got %v", i, test.out.instance, instance)
		}
	}
}

func TestSystemdUnitPath(t *testing.T) {

	var logg log.Logger = log.New(true)
	var st stage

	st.DestDir = "/"
	st.Logger = &logg

	tests := []struct {
		in  types.Unit
		out []string
	}{
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("system")},
			[]string{"etc/systemd/system"},
		},
		{
			types.Unit{Name: "test.service"},
			[]string{"etc/systemd/system"},
		},
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("global")},
			[]string{"etc/systemd/user"},
		},
	}

	for i, test := range tests {
		paths, err := st.SystemdUnitPaths(test.in)
		if err != nil {
			t.Errorf("Failed to get paths")
			t.FailNow()
		}
		if paths[len(paths)-1] != test.out[len(test.out)-1] {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, paths)
		}
	}
}

func TestSystemdDropinsPaths(t *testing.T) {

	var logg log.Logger = log.New(true)
	var st stage

	st.DestDir = "/"
	st.Logger = &logg

	tests := []struct {
		in  types.Unit
		out []string
	}{
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("system")},
			[]string{"etc/systemd/system/test.service.d"},
		},
		{
			types.Unit{Name: "test.service"},
			[]string{"etc/systemd/system/test.service.d"},
		},
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("global")},
			[]string{"etc/systemd/user/test.service.d"},
		},
	}

	for i, test := range tests {
		paths, err := st.SystemdDropinsPaths(test.in)
		if err != nil {
			t.Errorf("failed to get paths")
			t.FailNow()
		}
		if paths[len(paths)-1] != test.out[len(test.out)-1] {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, paths)
		}
	}
}

func TestSystemdPresetPath(t *testing.T) {

	var logg log.Logger = log.New(true)
	var st stage

	st.DestDir = "/"
	st.Logger = &logg

	tests := []struct {
		in  types.Unit
		out string
	}{
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("system")},
			"etc/systemd/system-preset/20-ignition.preset",
		},
		{
			types.Unit{Name: "test.service"},
			"etc/systemd/system-preset/20-ignition.preset",
		},
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("user")},
			"etc/systemd/user-preset/21-ignition.preset",
		},
		{
			types.Unit{Name: "test.service", Scope: cfgutil.StrToPtr("global")},
			"etc/systemd/user-preset/20-ignition.preset",
		},
	}

	for i, test := range tests {
		path := st.SystemdPresetPath(util.GetUnitScope(test.in))
		if path != test.out {
			t.Errorf("#%d: bad error: want %v, got %v", i, test.out, path)
		}
	}
}

func TestCreateUnits(t *testing.T) {

	if os.Geteuid() != 0 {
		t.Skip("test requires root for chroot(), skipping")
	}

	_, err := user.Lookup("root")
	if err != nil {
		t.Fatalf("user lookup failed (libnss_files.so might not be loaded): %v", err)
	}

	tmpdir, err := tempBase()
	if err != nil {
		t.Fatalf("temp base error: %v", err)
	}

	var logg log.Logger = log.New(true)
	var st stage

	err = st.checkRelabeling()

	if err != nil {
		t.Fatalf("checkRelabeling error: %v", err)
	}

	st.DestDir = tmpdir
	st.Logger = &logg

	defer os.RemoveAll(tmpdir)
	defer st.Logger.Close()

	var conf string = `{
		"ignition": {
			"version": "3.4.0-experimental"
		},
		"systemd": {
			"units": [
				{
					"contents": "[Unit]\nDescription=Prometheus node exporter\n[Install]\nWantedBy=multi-user.target\n",
					"enabled": true,
					"name": "unit1.service",
					"dropins": [{
						"name": "debug.conf",
						"contents": "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"
					  }]
				},
				{
					"contents": "[Unit]\nDescription=promtail.service\n[Install]\nWantedBy=multi-user.target default.target",
					"enabled": true,
					"name": "unit1.service",
					"scope": "user",
					"users" : ["tester1", "tester2"],
					"dropins": [{
						"name": "debug.conf",
						"contents": "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"
					  }]
				},
				{
					"contents": "[Unit]\nDescription=promtail.service\n[Install]\nWantedBy=multi-user.target default.target",
					"enabled": true,
					"name": "unit2.service",
					"scope": "system"
				},
				{
					"contents": "[Unit]\nDescription=promtail.service\n[Install]\nWantedBy=multi-user.target default.target",
					"enabled": true,
					"name": "unit2.service",
					"scope": "global"
				},
				{
					"contents": "[Unit]\nDescription=promtail.service\n[Install]\nWantedBy=multi-user.target default.target",
					"enabled": true,
					"name": "unit3.service",
					"scope": "global",
					"mask": true
				},
				{
					"contents": "[Unit]\nDescription=promtail.service\n[Install]\nWantedBy=multi-user.target default.target",
					"enabled": false,
					"name": "unit4.service",
					"scope": "user",
					"users" : ["tester1", "tester2"],
					"mask" : true,
					"dropins": [{
						"name": "debug.conf",
						"contents": "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"
					}]
				},
				{
					"contents": "[Unit]\nDescription=promtail.service\n[Install]\nWantedBy=multi-user.target default.target",
					"enabled": true,
					"name": "unit5.service",
					"scope": "global",
					"users" : ["tester1", "tester2"]
				}
			]
		}
	}`

	config, report, err := config.Parse([]byte(conf))

	if err != nil {
		fmt.Printf("error %v : \n%+v", err.Error(), report)
		t.FailNow()
	}
	fmt.Printf("validation report : \n%v", report)
	err = st.createUnits(config)
	if err != nil {
		t.Errorf("error occured: %v", err)
	}
}

func tempBase() (string, error) {
	td, err := ioutil.TempDir("", "igntests")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(td, "etc"), 0755); err != nil {
		return "", err
	}

	gp := filepath.Join(td, "etc/group")
	err = ioutil.WriteFile(gp, []byte("foo:x:4242:\n"), 0644)
	if err != nil {
		return "", err
	}

	pp := filepath.Join(td, "etc/passwd")
	err = ioutil.WriteFile(pp, []byte("tester1:x:44:4242::/home/tester1:/bin/false\ntester2:x:45:4242::/home/tester2:/bin/false"), 0644)
	if err != nil {
		return "", err
	}

	nsp := filepath.Join(td, "etc/nsswitch.conf")
	err = ioutil.WriteFile(nsp, []byte("passwd: files\ngroup: files\nshadow: files\ngshadow: files\n"), 0644)
	if err != nil {
		return "", err
	}

	return td, nil
}
