// Copyright 2016 CoreOS, Inc.
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

// The packet provider fetches a remote configuration from the packet.net
// userdata metadata service URL.

package packet

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/coreos/vcontext/report"
)

var (
	ErrValidFetchEmptyData = errors.New("fetch successful but fetched data empty")
)

var (
	userdataUrl = url.URL{
		Scheme: "https",
		Host:   "metadata.packet.net",
		Path:   "userdata",
	}
)

var (
	metadataUrl = url.URL{
		Scheme: "https",
		Host:   "metadata.packet.net",
		Path:   "metadata",
	}
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	// Packet's metadata service returns "Not Acceptable" when queried
	// with the default Accept header.
	headers := make(http.Header)
	headers.Set("Accept", "*/*")
	data, err := f.FetchToBuffer(userdataUrl, resource.FetchOptions{
		Headers: headers,
	})
	if err != nil {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}

// PostStatus posts a message that will show on the Packet Instance Timeline
func PostStatus(stageName string, f resource.Fetcher, errMsg error) error {
	f.Logger.Info("POST message to Packet Timeline")
	// fetch JSON from https://metadata.packet.net/metadata
	headers := make(http.Header)
	headers.Set("Accept", "*/*")
	data, err := f.FetchToBuffer(metadataUrl, resource.FetchOptions{
		Headers: headers,
	})
	if err != nil {
		return err
	}
	if data == nil {
		return ErrValidFetchEmptyData
	}
	metadata := struct {
		PhoneHomeURL string `json:"phone_home_url"`
	}{}
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return err
	}
	phonehomeURL := metadata.PhoneHomeURL
	// to get phonehome IPv4
	phonehomeURL = strings.TrimSuffix(phonehomeURL, "/phone-home")
	// POST Message to phonehome IP
	postMessageURL := phonehomeURL + "/events"

	return postMessage(stageName, errMsg, postMessageURL)
}

// postMessage makes a post request with the supplied message to the url
func postMessage(stageName string, e error, url string) error {

	stageName = "[" + stageName + "]"

	type mStruct struct {
		State   string `json:"state"`
		Message string `json:"message"`
	}
	var m mStruct
	if e != nil {
		m = mStruct{
			State:   "failed",
			Message: stageName + " Ignition error: " + e.Error(),
		}
	} else {
		m = mStruct{
			State:   "running",
			Message: stageName + " Ignition status: finished successfully",
		}
	}
	messageJSON, err := json.Marshal(m)
	if err != nil {
		return err
	}
	postReq, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJSON))
	if err != nil {
		return err
	}
	postReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	respPost, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer respPost.Body.Close()
	return err
}
