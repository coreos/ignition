// Copyright 2023 Red Hat
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

package applehv

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	"github.com/coreos/ignition/v2/internal/platform"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/vcontext/report"
	"github.com/mdlayher/vsock"
)

/*
	This provider is specific to virtual machines running under Apple Hypervisor on macOS on Apple hardware.

	It should however be possible to emulate the platform setup with QEMU, using [1] to assign a vsock to the
	guest and then forward the request from the Ignition process running in the virtual machine to an HTTP
	server running on the host, using the vsock support in socat for example.

	[1] https://wiki.qemu.org/Features/VirtioVsock
*/

func init() {
	platform.Register(platform.Provider{
		Name:  "applehv",
		Fetch: fetchConfig,
	})
}

func fetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	// the vsock module must be built into the kernel or loaded so we can communicate
	// with the host
	if _, statErr := os.Stat("/sys/devices/virtual/misc/vsock"); statErr != nil {
		if _, err := f.Logger.LogCmd(exec.Command(distro.ModprobeCmd(), "vsock"), "Loading vsock kernel module"); err != nil {
			f.Logger.Err("failed to install vsock kernel module: %v", err)
			return types.Config{}, report.Report{}, fmt.Errorf("failed to install vsock kernel module: %v", err)
		}
	}

	// we use a http GET over vsock to fetch the ignition file.  the
	// vsock connection itself is begun here. the "host" will need an HTTPD
	// server listen on the the other end of the vsock connection on port 1024. The
	// port is trivial and was just chosen by author
	// ID =2 is shorthand for "the host"
	//
	conn, err := vsock.Dial(2, 1024, &vsock.Config{})
	if err != nil {
		return types.Config{}, report.Report{}, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			f.Logger.Err("unable to close vsock connection: %v", err)
		}
	}()

	// The host portion of the URL is arbitrary here.  The schema is important however.  Because
	// this is more or less HTTP over a UDS, then the host name is discarded.
	req, err := http.NewRequest(http.MethodGet, "http://d/", nil)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return conn, nil
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			f.Logger.Err("unable to close response body: %v", err)
		}
	}()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, b)
}
