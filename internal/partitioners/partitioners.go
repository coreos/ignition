package partitioners

import (
	"github.com/coreos/ignition/v2/config/v3_5_experimental/types"
)

type DeviceManager interface {
	CreatePartition(p Partition)
	DeletePartition(num int)
	Info(num int)
	WipeTable(wipe bool)
	Pretend() (string, error)
	Commit() error
	ParseOutput(string, []int) (map[int]Output, error)
}

type Partition struct {
	types.Partition
	StartSector   *int64
	SizeInSectors *int64
	StartMiB      string
	SizeMiB       string
}

type Output struct {
	Start int64
	Size  int64
}
