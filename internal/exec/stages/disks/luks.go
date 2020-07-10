// Copyright 2020 Red Hat
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

package disks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
)

// https://github.com/latchset/clevis/blob/master/src/pins/tang/clevis-encrypt-tang.1.adoc#config
type Tang struct {
	URL        string `json:"url"`
	Thumbprint string `json:"thp,omitempty"`
}

// https://github.com/latchset/clevis/blob/master/README.md#pin-shamir-secret-sharing
type Pin struct {
	Tpm  bool
	Tang []Tang
}

func (p Pin) MarshalJSON() ([]byte, error) {
	if p.Tpm {
		return json.Marshal(&struct {
			Tang []Tang   `json:"tang,omitempty"`
			Tpm  struct{} `json:"tpm2"`
		}{
			Tang: p.Tang,
			Tpm:  struct{}{},
		})
	} else {
		return json.Marshal(&struct {
			Tang []Tang `json:"tang"`
		}{
			Tang: p.Tang,
		})
	}
}

type Clevis struct {
	Pins      Pin `json:"pins"`
	Threshold int `json:"t"`
}

// Initially tested generating keyfiles via dd'ing to a file from /dev/urandom
// however while cryptsetup had no problem with these keyfiles clevis seemed to
// die on them while keyfiles generated via openssl rand -hex would work...
func randHex(length int) (string, error) {
	bytes := make([]byte, length)
	// On older kernels this could block indefinitely but there's nothing
	// that we can do about it; we don't want to use earlyrand
	// https://lwn.net/Articles/802360/
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *stage) createLuks(config types.Config) error {
	if len(config.Storage.Luks) == 0 {
		return nil
	}
	s.Logger.PushPrefix("createLuks")
	defer s.Logger.PopPrefix()

	for _, luks := range config.Storage.Luks {
		// TODO: allow Ignition generated KeyFiles for
		// non-clevis devices that can be persisted.
		// TODO: create devices in parallel.
		// track whether Ignition creates the KeyFile
		// so that it can be removed
		var ignitionCreatedKeyFile bool
		// create keyfile inside of tmpfs, it will be copied to the
		// sysroot by the files stage
		os.MkdirAll(distro.LuksInitramfsKeyFilePath(), 0700)
		keyFilePath := filepath.Join(distro.LuksInitramfsKeyFilePath(), luks.Name)
		if util.NilOrEmpty(luks.KeyFile.Source) {
			// create a keyfile
			key, err := randHex(4096)
			if err != nil {
				return fmt.Errorf("generating keyfile: %v", err)
			}
			if err := ioutil.WriteFile(keyFilePath, []byte(key), 0400); err != nil {
				return fmt.Errorf("creating keyfile: %v", err)
			}
			ignitionCreatedKeyFile = true
		} else {
			f := types.File{
				Node: types.Node{
					Path: keyFilePath,
				},
				FileEmbedded1: types.FileEmbedded1{
					Contents: luks.KeyFile,
				},
			}
			fetchOps, err := s.Util.PrepareFetches(s.Util.Logger, f)
			if err != nil {
				return fmt.Errorf("failed to resolve keyfile %q: %v", f.Path, err)
			}
			for _, op := range fetchOps {
				if err := s.Util.Logger.LogOp(
					func() error {
						return s.Util.PerformFetch(op)
					}, "writing file %q", f.Path,
				); err != nil {
					return fmt.Errorf("failed to create keyfile %q: %v", op.Node.Path, err)
				}
			}
		}
		args := []string{
			"luksFormat",
			"--type", "luks2",
			"--key-file", keyFilePath,
		}

		if !util.NilOrEmpty(luks.Label) {
			args = append(args, "--label", *luks.Label)
		}

		if !util.NilOrEmpty(luks.UUID) {
			args = append(args, "--uuid", *luks.UUID)
		}

		if len(luks.Options) > 0 {
			// golang's a really great language...
			for _, option := range luks.Options {
				args = append(args, string(option))
			}
		}

		args = append(args, *luks.Device)

		if _, err := s.Logger.LogCmd(
			exec.Command(distro.CryptsetupCmd(), args...),
			"creating %q", luks.Name,
		); err != nil {
			return fmt.Errorf("cryptsetup failed: %v", err)
		}

		// open the device
		if _, err := s.Logger.LogCmd(
			exec.Command(distro.CryptsetupCmd(), "luksOpen", *luks.Device, luks.Name, "--key-file", keyFilePath),
			"opening luks device %v", luks.Name,
		); err != nil {
			return fmt.Errorf("opening luks device: %v", err)
		}

		if luks.Clevis != nil {
			c := Clevis{
				Threshold: 1,
			}
			if luks.Clevis.Threshold != nil {
				c.Threshold = *luks.Clevis.Threshold
			}
			for _, tang := range luks.Clevis.Tang {
				c.Pins.Tang = append(c.Pins.Tang, Tang{
					URL:        tang.URL,
					Thumbprint: *tang.Thumbprint,
				})
			}
			if luks.Clevis.Tpm2 != nil {
				c.Pins.Tpm = *luks.Clevis.Tpm2
			}
			clevisJson, err := json.Marshal(c)
			if err != nil {
				return fmt.Errorf("creating clevis json: %v", err)
			}
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.ClevisCmd(), "luks", "bind", "-f", "-k", keyFilePath, "-d", *luks.Device, "sss", string(clevisJson)), "Clevis bind",
			); err != nil {
				return fmt.Errorf("binding clevis device: %v", err)
			}

			// close & re-open Clevis devices to make sure that we can unlock them
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.CryptsetupCmd(), "luksClose", luks.Name),
				"closing clevis luks device %v", luks.Name,
			); err != nil {
				return fmt.Errorf("closing luks device: %v", err)
			}
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.ClevisCmd(), "luks", "unlock", "-d", *luks.Device, "-n", luks.Name),
				"reopening clevis luks device %s", luks.Name,
			); err != nil {
				return fmt.Errorf("reopening luks device %s: %v", luks.Name, err)
			}
		}

		// assume the user does not want a key file, remove it
		if ignitionCreatedKeyFile {
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.CryptsetupCmd(), "luksRemoveKey", *luks.Device, keyFilePath),
				"removing key file for %v", luks.Name,
			); err != nil {
				return fmt.Errorf("removing key file from luks device: %v", err)
			}
			if err := os.Remove(keyFilePath); err != nil {
				return fmt.Errorf("removing key file: %v", err)
			}
		}
	}

	return nil
}
