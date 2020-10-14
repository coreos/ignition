// Copyright 2017 CoreOS, Inc.
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

package passwd

import (
	"github.com/coreos/ignition/v2/tests/register"
	"github.com/coreos/ignition/v2/tests/types"
)

func init() {
	register.Register(register.PositiveTest, AddPasswdUsers())
	register.Register(register.PositiveTest, DeletePasswdUsers())
	register.Register(register.PositiveTest, DeleteGroups())
	register.Register(register.PositiveTest, UseAuthorizedKeysFile())
}

func AddPasswdUsers() types.Test {
	name := "users.add"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	env := []string{"IGNITION_WRITE_AUTHORIZED_KEYS_FRAGMENT=true"}
	config := `{
		"ignition": {
			"version": "$version"
		},
		"passwd": {
			"users": [{
					"name": "test",
					"passwordHash": "zJW/EKqqIk44o",
					"sshAuthorizedKeys": [
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDBRZPFJNOvQRfokigTtl0IBi71LHZrFOk4EJ3Zowtk/bX5uIVai0Cd4+hqlocYL10idgtFBH28skeKfsmHwgS9XwOvP+g+kqAl7yCz8JEzIUzl1fxNZDToi0jA3B5MwXkpt+IWfnabwi2cRZhlzrz9rO+eExu5s3NfaRmmmCYrjCJIRPKSCrW8U0n9fVSbX4PDdMXVmH7r+t8MtR8523vCbakFR/Y0YIqkPVdfuUXHh9rDCdH4B7mt7nYX2LWQXGUvmI13mgQoy04ifkaR3ImuOMp3Y1J1gm6clO74IMCq/sK9+XJhbxMPPHUoUJ2EwbaG7Dbh3iqz47e9oVki4gIH stephenlowrie@localhost.localdomain"
					]
				},
				{
					"name": "jenkins",
					"uid": 1020,
					"shouldExist": true
				}
			]
		}
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\n",
		},
		{
			Node: types.Node{
				Name:      "shadow",
				Directory: "etc",
			},
			Contents: "root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\n",
		},
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\n",
		},
		{
			Node: types.Node{
				Name:      "nsswitch.conf",
				Directory: "etc",
			},
			Contents: "# /etc/nsswitch.conf:\n\npasswd:      files\nshadow:      files\ngroup:       files\n\nhosts:       files dns myhostname\nnetworks:    files dns\n\nservices:    files\nprotocols:   files\nrpc:         files\n\nethers:      files\nnetmasks:    files\nnetgroup:    files\nbootparams:  files\nautomount:   files\naliases:     files\n",
		},
		{
			Node: types.Node{
				Name:      "login.defs",
				Directory: "etc",
			},
			Contents: `MAIL_DIR	/var/spool/mail
PASS_MAX_DAYS	99999
PASS_MIN_DAYS	0
PASS_MIN_LEN	5
PASS_WARN_AGE	7
UID_MIN                  1000
UID_MAX                 60000
SYS_UID_MIN               201
SYS_UID_MAX               999
GID_MIN                  1000
GID_MAX                 60000
SYS_GID_MIN               201
SYS_GID_MAX               999
CREATE_HOME	yes
UMASK           077
USERGROUPS_ENAB yes
ENCRYPT_METHOD SHA512
`,
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\ntest:x:1000:1000::/home/test:/bin/bash\njenkins:x:1020:1001::/home/jenkins:/bin/bash\n",
		},
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\ntest:x:1000:\njenkins:x:1001:\n",
		},
		{
			Node: types.Node{
				Name:      "shadow",
				Directory: "etc",
			},
			Contents: "root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\ntest:zJW/EKqqIk44o:17331:0:99999:7:::\njenkins:*:17331:0:99999:7:::\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\ntest:!::\njenkins:!::\n",
		},
		{
			Node: types.Node{
				Name:      "ignition",
				Directory: "home/test/.ssh/authorized_keys.d",
				User:      1000,
				Group:     1000,
			},
			Contents: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDBRZPFJNOvQRfokigTtl0IBi71LHZrFOk4EJ3Zowtk/bX5uIVai0Cd4+hqlocYL10idgtFBH28skeKfsmHwgS9XwOvP+g+kqAl7yCz8JEzIUzl1fxNZDToi0jA3B5MwXkpt+IWfnabwi2cRZhlzrz9rO+eExu5s3NfaRmmmCYrjCJIRPKSCrW8U0n9fVSbX4PDdMXVmH7r+t8MtR8523vCbakFR/Y0YIqkPVdfuUXHh9rDCdH4B7mt7nYX2LWQXGUvmI13mgQoy04ifkaR3ImuOMp3Y1J1gm6clO74IMCq/sK9+XJhbxMPPHUoUJ2EwbaG7Dbh3iqz47e9oVki4gIH stephenlowrie@localhost.localdomain\n",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Env:              env,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

// DeletePasswdUsers verifies that the user(s) can be removed
// from a given OS distro.
func DeletePasswdUsers() types.Test {
	name := "users.delete"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "$version"
		},
		"passwd": {
			"users": [
				{
					"name": "jenkins",
					"shouldExist": false
				},
				{
					"name": "test",
					"shouldExist": false
				}
			]
		}
	}`
	configMinVersion := "3.2.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\njenkins:x:1020:1001::/:/bin/bash\n",
		},
		{
			Node: types.Node{
				Name:      "shadow",
				Directory: "etc",
			},
			Contents: "root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\njenkins:*:17331:0:99999:7:::\n",
		},
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\njenkins:x:1001:\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\njenkins:!::\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\n",
		},
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\n",
		},
		{
			Node: types.Node{
				Name:      "shadow",
				Directory: "etc",
			},
			Contents: "root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\n",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

// DeleteGroups verifies that the group(s) can be removed
// from a given OS distro.
func DeleteGroups() types.Test {
	name := "groups.delete"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	config := `{
		"ignition": {
			"version": "$version"
		},
		"passwd": {
			"groups": [
				{
					"name": "jenkins",
					"shouldExist": false
				},
				{
					"name": "test",
					"shouldExist": false
				}
			]
		}
	}`
	configMinVersion := "3.2.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\njenkins:x:1020:1001::/:/bin/bash\n",
		},
		{
			Node: types.Node{
				Name:      "shadow",
				Directory: "etc",
			},
			Contents: "root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\njenkins:*:17331:0:99999:7:::\n",
		},
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\njenkins:x:1001:\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\njenkins:!::\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\n",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}

// UseAuthorizedKeysFile verifies that ~/.ssh/authorized_keys is written
// when IGNITION_WRITE_AUTHORIZED_KEYS_FRAGMENT=false.
func UseAuthorizedKeysFile() types.Test {
	name := "users.authorized_keys"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	env := []string{"IGNITION_WRITE_AUTHORIZED_KEYS_FRAGMENT=false"}
	config := `{
		"ignition": {
			"version": "$version"
		},
		"passwd": {
			"users": [{
					"name": "test",
					"passwordHash": "zJW/EKqqIk44o",
					"sshAuthorizedKeys": [
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDBRZPFJNOvQRfokigTtl0IBi71LHZrFOk4EJ3Zowtk/bX5uIVai0Cd4+hqlocYL10idgtFBH28skeKfsmHwgS9XwOvP+g+kqAl7yCz8JEzIUzl1fxNZDToi0jA3B5MwXkpt+IWfnabwi2cRZhlzrz9rO+eExu5s3NfaRmmmCYrjCJIRPKSCrW8U0n9fVSbX4PDdMXVmH7r+t8MtR8523vCbakFR/Y0YIqkPVdfuUXHh9rDCdH4B7mt7nYX2LWQXGUvmI13mgQoy04ifkaR3ImuOMp3Y1J1gm6clO74IMCq/sK9+XJhbxMPPHUoUJ2EwbaG7Dbh3iqz47e9oVki4gIH stephenlowrie@localhost.localdomain"
					]
				}
			]
		}
	}`
	configMinVersion := "3.0.0"
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "passwd",
				Directory: "etc",
			},
			Contents: "root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\n",
		},
		{
			Node: types.Node{
				Name:      "shadow",
				Directory: "etc",
			},
			Contents: "root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\n",
		},
		{
			Node: types.Node{
				Name:      "group",
				Directory: "etc",
			},
			Contents: "root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\n",
		},
		{
			Node: types.Node{
				Name:      "gshadow",
				Directory: "etc",
			},
			Contents: "root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\n",
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "authorized_keys",
				Directory: "home/test/.ssh",
				User:      1000,
				Group:     1000,
			},
			Contents: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDBRZPFJNOvQRfokigTtl0IBi71LHZrFOk4EJ3Zowtk/bX5uIVai0Cd4+hqlocYL10idgtFBH28skeKfsmHwgS9XwOvP+g+kqAl7yCz8JEzIUzl1fxNZDToi0jA3B5MwXkpt+IWfnabwi2cRZhlzrz9rO+eExu5s3NfaRmmmCYrjCJIRPKSCrW8U0n9fVSbX4PDdMXVmH7r+t8MtR8523vCbakFR/Y0YIqkPVdfuUXHh9rDCdH4B7mt7nYX2LWQXGUvmI13mgQoy04ifkaR3ImuOMp3Y1J1gm6clO74IMCq/sK9+XJhbxMPPHUoUJ2EwbaG7Dbh3iqz47e9oVki4gIH stephenlowrie@localhost.localdomain\n",
		},
	})

	return types.Test{
		Name:             name,
		In:               in,
		Out:              out,
		Env:              env,
		Config:           config,
		ConfigMinVersion: configMinVersion,
	}
}
