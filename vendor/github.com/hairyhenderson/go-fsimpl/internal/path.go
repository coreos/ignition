package internal

import (
	"io/fs"
	"strings"
)

func ValidPath(name string) bool {
	if strings.Contains(name, "\\") {
		return false
	}

	return fs.ValidPath(name)
}
