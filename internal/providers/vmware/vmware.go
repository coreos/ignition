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

// The vmware provider fetches a configuration from the VMware Guest Info
// interface.

package vmware

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/coreos/ignition/internal/log"
	"github.com/coreos/ignition/internal/providers"
	"github.com/coreos/ignition/internal/util"
)

type Creator struct{}

func (Creator) Create(logger *log.Logger, _ util.HttpClient) providers.Provider {
	return &provider{
		logger: logger,
	}
}

type provider struct {
	logger *log.Logger
}

func (p provider) ShouldRetry() bool {
	return false
}

func (p *provider) BackoffDuration() time.Duration {
	return 0
}

func decodeData(data string, encoding string) ([]byte, error) {
	switch encoding {
	case "":
		return []byte(data), nil

	case "b64", "base64":
		return decodeBase64Data(data)

	case "gz", "gzip":
		return decodeGzipData(data)

	case "gz+base64", "gzip+base64", "gz+b64", "gzip+b64":
		gz, err := decodeBase64Data(data)

		if err != nil {
			return nil, err
		}

		return decodeGzipData(string(gz))
	}

	return nil, fmt.Errorf("Unsupported encoding %q", encoding)
}

func decodeBase64Data(data string) ([]byte, error) {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode base64: %q", err)
	}

	return decodedData, nil
}

func decodeGzipData(data string) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}
