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

// The aws provider fetches a remote configuration from the EC2 user-data
// metadata service URL.

package aws

import (
	"net/url"

	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/providers/util"
	"github.com/coreos/ignition/v2/internal/resource"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/coreos/vcontext/report"
)

var (
	userdataUrl = url.URL{
		Scheme: "http",
		Host:   "169.254.169.254",
		Path:   "2019-10-01/user-data",
	}
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := f.FetchToBuffer(userdataUrl, resource.FetchOptions{})
	if err != nil && err != resource.ErrNotFound {
		return types.Config{}, report.Report{}, err
	}

	return util.ParseConfig(f.Logger, data)
}

func NewFetcher(l *log.Logger) (resource.Fetcher, error) {
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return resource.Fetcher{}, err
	}
	sess.Config.Credentials = ec2rolecreds.NewCredentials(sess)

	return resource.Fetcher{
		Logger:     l,
		AWSSession: sess,
	}, nil
}

func Init(f *resource.Fetcher) error {
	// Determine the partition and region this instance is in
	regionHint, err := ec2metadata.New(f.AWSSession).Region()
	if err != nil {
		regionHint = "us-east-1"
	}
	f.S3RegionHint = regionHint
	return nil
}
