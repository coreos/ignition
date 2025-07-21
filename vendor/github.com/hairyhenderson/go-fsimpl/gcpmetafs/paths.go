package gcpmetafs

import (
	"regexp"
	"strings"
)

// With the GCP VM Metadata Service, there's no generic way to know whether a
// GET request has returned a document or a directory. However, the structure of
// the metadata service is documented, so we can hard-code the known directory
// paths.
//
// The main directories are:
// - /instance: instance metadata (specific to the VM)
// - /project: project metadata (shared across all VMs in the project)
//
// Within these paths, data is organized into categories. For example:
// - /instance/id: The VM's unique identifier
// - /instance/hostname: The VM's hostname
// - /instance/network-interfaces/: Network interface information
// - /project/project-id: The project ID
//
// Another option could be to build the list dynamically by querying the
// paths (the responses indicate which children are directories - they have
// a "/" suffix), but that would require additional requests.
//
// For more information, see:
// https://cloud.google.com/compute/docs/metadata/default-metadata-values
func isMetadataDirectory(name string) bool {
	name = strings.Trim(name, "/")

	if isRootOrMainCategory(name) {
		return true
	}

	if isKnownInstanceSubdirectory(name) {
		return true
	}

	if isPathBasedDirectory(name) {
		return true
	}

	if name == "project/attributes" {
		return true
	}

	return false
}

// isRootOrMainCategory checks if the path is a root or main category directory.
// These are the top-level directories in the metadata service.
func isRootOrMainCategory(name string) bool {
	if name == "" || name == "." {
		return true
	}

	if name == "instance" || name == "project" {
		return true
	}

	return false
}

// isKnownInstanceSubdirectory checks if the path is a known instance subdirectory.
// These are directories that contain metadata about the instance.
func isKnownInstanceSubdirectory(name string) bool {
	switch name {
	case "instance/attributes",
		"instance/disks",
		"instance/network-interfaces",
		"instance/service-accounts",
		"instance/tags",
		"instance/scheduling",
		"instance/licenses":
		return true
	}

	// Check for service account subdirectories
	if strings.HasPrefix(name, "instance/service-accounts/") {
		parts := strings.Split(name, "/")
		if len(parts) == 3 {
			// This is a service account directory like "instance/service-accounts/default"
			return true
		}
	}

	return false
}

// isPathBasedDirectory checks if the path matches one of the regex patterns
// that identify directories in the metadata service.
func isPathBasedDirectory(name string) bool {
	// Check against all regex patterns
	patterns := []*regexp.Regexp{
		diskDirectoryRe,
		networkInterfaceDirectoryRe,
		accessConfigsDirectoryRe,
		accessConfigDirectoryRe,
		forwardedIPsDirectoryRe,
		serviceAccountDirectoryRe,
	}

	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}

	return false
}

var (
	diskDirectoryRe             = regexp.MustCompile(`^instance/disks/\d+$`)
	networkInterfaceDirectoryRe = regexp.MustCompile(`^instance/network-interfaces/\d+$`)
	accessConfigsDirectoryRe    = regexp.MustCompile(`^instance/network-interfaces/\d+/access-configs$`)
	accessConfigDirectoryRe     = regexp.MustCompile(`^instance/network-interfaces/\d+/access-configs/\d+$`)
	forwardedIPsDirectoryRe     = regexp.MustCompile(`^instance/network-interfaces/\d+/forwarded-ips$`)
	serviceAccountDirectoryRe   = regexp.MustCompile(`^instance/service-accounts/[^/]+$`)
)
