package util

import (
	"bytes"
)

func IsCloudConfig(contents []byte) bool {
	header := []byte("#cloud-config\n")
	if bytes.HasPrefix(contents, header) {
		return true
	}
	return false
}
