package util

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/coreos/ignition/config/types"
)

// BlockDeviceInfo information extracted from blkid command
type BlockDeviceInfo struct {
	Name  string
	Label string
	Type  string
}

// BlockDevice returns the BlockDeviceInfo for a given device
func BlockDevice(dev types.Path) (*BlockDeviceInfo, error) {
	args := []string{
		"-o", "export",
		string(dev),
	}

	command := exec.Command("blkid", args...)
	output, err := command.Output()
	if err != nil && len(output) != 0 {
		return nil, fmt.Errorf("error retrieving %q information: %s", dev, err)
	}

	if len(output) == 0 {
		return nil, nil
	}

	return parseBlkidOutput(string(output)), nil
}

func parseBlkidOutput(stdout string) *BlockDeviceInfo {
	blk := &BlockDeviceInfo{}

	for _, line := range strings.Split(stdout, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}

		switch parts[0] {
		case "DEVNAME":
			blk.Name = parts[1]
		case "LABEL":
			blk.Label = parts[1]
		case "TYPE":
			blk.Type = parts[1]
		}
	}

	return blk
}
