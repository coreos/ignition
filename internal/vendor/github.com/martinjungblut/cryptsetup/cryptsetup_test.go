package cryptsetup

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

const PASSKEY string = "pkey"
const VOLUME_NAME string = "testdata"

func TestInit(t *testing.T) {
	err, _ := Init(VOLUME_NAME)
	if err != nil {
		t.Error(fmt.Sprintf("Init() raised an error: %s", err.Error()))
	}
}

func TestInitFails(t *testing.T) {
	err, _ := Init("non_existing_device")
	if err == nil {
		t.Error(fmt.Sprintf("Init() did not raise an error!"))
	}
}

func TestFormatLUKS (t *testing.T) {
	err, device := Init(VOLUME_NAME)
	if err != nil {
		t.Error(fmt.Sprintf("Init() raised an error: %s", err.Error()))
	}

	params := LUKSParams{hash: "sha1", data_alignment: 0, data_device: ""}

	err = device.FormatLUKS("aes", "xts-plain64", "", "", 256 / 8, params)
	if err != nil {
		t.Error(fmt.Sprintf("FormatLUKS() raised an error: %s", err.Error()))
	}
}

func setup() {
	exec.Command("/bin/dd", "if=/dev/zero", fmt.Sprintf("of=%s", VOLUME_NAME), "bs=4M", "count=1").Run()
}

func teardown() {
	exec.Command("/bin/rm", "-rf", VOLUME_NAME).Run()
}

func TestMain (m *testing.M) {
	setup()
	result := m.Run()
	teardown()
	os.Exit(result)
}
