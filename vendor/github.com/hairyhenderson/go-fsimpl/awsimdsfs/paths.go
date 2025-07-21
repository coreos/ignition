package awsimdsfs

import (
	"regexp"
	"strings"
)

// With the IMDS API, there's no generic way to know whether a GET request has
// returned a document or a directory; the content-type is no help, and paths
// can be queried with or without a trailing slash. However, the full list of
// paths returned by the IMDS API is documented here:
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-categories.html,
// so we can just hard-code that list here for now.
//
// Another option could be to build the list dynamically by querying the
// meta-data/ and dynamic/ paths (the responses indicate which children are
// files vs directories - they have a "/" suffix), but that would be very
// inefficient.
func isIMDSDirectory(name string) bool {
	name = strings.Trim(name, "/")

	// user-data is a file, not a directory
	if name == "user-data" {
		return false
	}

	switch name {
	case "meta-data",
		"dynamic",
		"meta-data/autoscaling",
		"meta-data/block-device-mapping",
		"meta-data/events",
		"meta-data/events/maintenance",
		"meta-data/events/recommendations",
		"meta-data/iam",
		"meta-data/iam/security-credentials",
		"meta-data/identity-credentials",
		"meta-data/identity-credentials/ec2",
		"meta-data/identity-credentials/ec2/security-credentials",
		"meta-data/metrics",
		"meta-data/network",
		"meta-data/network/interfaces",
		"meta-data/network/interfaces/macs",
		"meta-data/placement",
		"meta-data/public-keys",
		"meta-data/public-keys/0",
		"meta-data/services",
		"meta-data/spot",
		"meta-data/tags",
		"dynamic/fws",
		"dynamic/instance-identity":
		return true
	}

	// the <mac> part is variable, but both of these are directories:
	// - network/interfaces/macs/<mac>/
	// - network/interfaces/macs/<mac>/ipv4-associations/
	// this however is not:
	// - network/interfaces/macs/<mac>/ipv4-associations/<ip>
	if macDirectoryRe.MatchString(name) {
		return true
	}

	if macIPAssociationDirectoryRe.MatchString(name) {
		return true
	}

	return false
}

var (
	macDirectoryRe              = regexp.MustCompile(`^network/interfaces/macs/[^/]+$`)
	macIPAssociationDirectoryRe = regexp.MustCompile(`^network/interfaces/macs/[^/]+/ipv4-associations$`)
)
