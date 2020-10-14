// Copyright 2020 Red Hat, Inc.
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

package fetch_offline

import (
	"testing"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"

	"github.com/stretchr/testify/assert"
)

func checkNeedsNet(t *testing.T, cfg *types.Config) bool {
	needsNet, err := configNeedsNet(cfg)
	assert.Equal(t, err, nil, "unexpected error: %v", err)
	return needsNet
}

func assertNeedsNet(t *testing.T, cfg *types.Config) {
	assert.Equal(t, checkNeedsNet(t, cfg), true, "unexpected config doesn't need net: %v", *cfg)
}

func assertNotNeedsNet(t *testing.T, cfg *types.Config) {
	assert.Equal(t, checkNeedsNet(t, cfg), false, "unexpected config needs net: %v", *cfg)
}

func TestConfigNeedsNet(t *testing.T) {
	cfg := types.Config{
		Ignition: types.Ignition{
			Version: "3.3.0-experimental",
			Config: types.IgnitionConfig{
				Replace: types.Resource{
					Source: util.StrToPtr("http://example.com/config.ign"),
				},
			},
		},
	}
	assertNeedsNet(t, &cfg)
	cfg.Ignition.Config.Replace.Source = nil
	assertNotNeedsNet(t, &cfg)
	cfg.Storage.Files = []types.File{
		{
			Node: types.Node{
				Path: "/etc/foobar.conf",
			},
			FileEmbedded1: types.FileEmbedded1{
				Contents: types.Resource{
					Source: util.StrToPtr("data:,hello"),
				},
			},
		},
	}
	assertNotNeedsNet(t, &cfg)
	cfg.Storage.Files[0].Contents.Source = util.StrToPtr("")
	assertNotNeedsNet(t, &cfg)
	cfg.Storage.Files[0].Contents.Source = nil
	assertNotNeedsNet(t, &cfg)
	cfg.Storage.Files[0].Contents.Source = util.StrToPtr("http://example.com/payload")
	assertNeedsNet(t, &cfg)
	cfg.Storage.Files[0].Contents.Source = nil
	assertNotNeedsNet(t, &cfg)
	cfg.Storage.Luks = []types.Luks{
		{
			Name:   "foobar",
			Device: util.StrToPtr("bazboo"),
			Clevis: &types.Clevis{
				Tang: []types.Tang{
					{
						Thumbprint: util.StrToPtr("mythumbprint"),
						URL:        "http://tang.example.com",
					},
				},
			},
		},
	}
	assertNeedsNet(t, &cfg)
}
