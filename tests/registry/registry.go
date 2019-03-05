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

package registry

// Done to import the tests
import (
	_ "github.com/coreos/ignition/v2/tests/negative/files"
	_ "github.com/coreos/ignition/v2/tests/negative/filesystems"
	_ "github.com/coreos/ignition/v2/tests/negative/general"
	_ "github.com/coreos/ignition/v2/tests/negative/partitions"
	_ "github.com/coreos/ignition/v2/tests/negative/proxy"
	_ "github.com/coreos/ignition/v2/tests/negative/regression"
	_ "github.com/coreos/ignition/v2/tests/negative/security"
	_ "github.com/coreos/ignition/v2/tests/negative/timeouts"
	_ "github.com/coreos/ignition/v2/tests/positive/files"
	_ "github.com/coreos/ignition/v2/tests/positive/filesystems"
	_ "github.com/coreos/ignition/v2/tests/positive/general"
	_ "github.com/coreos/ignition/v2/tests/positive/partitions"
	_ "github.com/coreos/ignition/v2/tests/positive/passwd"
	_ "github.com/coreos/ignition/v2/tests/positive/proxy"
	_ "github.com/coreos/ignition/v2/tests/positive/regression"
	_ "github.com/coreos/ignition/v2/tests/positive/security"
	_ "github.com/coreos/ignition/v2/tests/positive/systemd"
	_ "github.com/coreos/ignition/v2/tests/positive/timeouts"
)
