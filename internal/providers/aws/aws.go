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
	"errors"
	"net/http"
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
	userdataURL = url.URL{
		Scheme: "http",
		Host:   "169.254.169.254",
		Path:   "2019-10-01/user-data",
	}
	imdsTokenURL = url.URL{
		Scheme: "http",
		Host:   "169.254.169.254",
		Path:   "latest/api/token",
	}
	errIMDSV2 = errors.New("failed to fetch IMDSv2 session token")
)

func FetchConfig(f *resource.Fetcher) (types.Config, report.Report, error) {
	data, err := fetchFromAWSMetadata(userdataURL, resource.FetchOptions{}, f)
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

// Init prepares the fetcher for this platform
func Init(f *resource.Fetcher) error {
	// Determine the partition and region this instance is in
	regionHint, err := ec2metadata.New(f.AWSSession).Region()
	if err != nil {
		regionHint = "us-east-1"
		f.Logger.Warning("failed to determine EC2 region, falling back to default %s: %v", regionHint, err)
	}
	f.S3RegionHint = regionHint
	return nil
}

// fetchFromAWSMetadata fetches metadata from the `IMDSv2` service if its
// configured, else it will fall back to `IMDSv1`.
func fetchFromAWSMetadata(u url.URL, opts resource.FetchOptions, f *resource.Fetcher) ([]byte, error) {
	token, err := fetchAWSIMDSV2Token(f)
	if err == errIMDSV2 {
		// Do nothing
		f.Logger.Info("IMDSv2 service is unavailable; falling back to IMDSv1")
	} else if err != nil {
		return nil, err
	} else {
		opts.Headers.Add("X-aws-ec2-metadata-token", token)
	}
	return f.FetchToBuffer(u, opts)
}

// fetchAWSIMDSV2Token fetches a session token from an EC2 instance (if the
// instace is configured to use `IMDSv2`), otherwise, it will return an error.
func fetchAWSIMDSV2Token(f *resource.Fetcher) (string, error) {
	opts := resource.FetchOptions{
		Headers: http.Header{
			"x-aws-ec2-metadata-token-ttl-seconds": []string{"21600"},
		},
		HTTPVerb: "PUT",
	}
	token, err := f.FetchToBuffer(imdsTokenURL, opts)
	if err == resource.ErrNotFound {
		f.Logger.Debug("cannot read IMDSv2 session token from %q", imdsTokenURL.String())
		return "", errIMDSV2
	} else if err != nil {
		f.Logger.Debug("unexpected error retrieving IMDSv2 session token: %v", err)
		return "", err
	}
	return string(token), nil
}
