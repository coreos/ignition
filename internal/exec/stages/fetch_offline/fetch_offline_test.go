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
	"github.com/coreos/ignition/v2/config/v3_7_experimental/types"

	"github.com/stretchr/testify/assert"
)

func checkNeedsNet(t *testing.T, cfg *types.Config) bool {
	needsNet, err := configNeedsNet(cfg)
	assert.Equal(t, err, nil, "unexpected error: %v", err)
	return needsNet
}

func TestConfigNotNeedsNet(t *testing.T) {
	tests := []types.Config{
		// Test ClevisCustom NeedsNetwork set to false
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("foo"),
						Clevis: types.Clevis{
							Custom: types.ClevisCustom{
								NeedsNetwork: util.BoolToPtr(false),
							},
						},
					},
				},
			},
		},
		// Source with no URL
		{
			Ignition: types.Ignition{
				Version: "3.3.0",
				Config: types.IgnitionConfig{
					Replace: types.Resource{
						Source: util.StrToPtr(""),
					},
				},
			},
		},
		// Source with Nil
		{
			Ignition: types.Ignition{
				Version: "3.3.0",
				Config: types.IgnitionConfig{
					Replace: types.Resource{
						Source: nil,
					},
				},
			},
		},
		// Empty Config
		{},
		// Tang with adv set does not need networking on first boot.
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("foo"),
						Clevis: types.Clevis{
							Tang: []types.Tang{
								{
									Thumbprint:    util.StrToPtr("mythumbprint"),
									URL:           "http://mytang.example.com",
									Advertisement: util.StrToPtr(" {\"payload\": \"...\",\"protected\":\"...\",\"signature\":\"...\"}"),
								},
							},
						},
					},
				},
			},
		},
	}

	for i, test := range tests {
		assert.Equal(t, checkNeedsNet(t, &test), false, "#%d: unexpected config needs net: %v", i, test)
	}
}

func TestConfigNeedsNet(t *testing.T) {
	tests := []types.Config{
		// Tang with no adv set needs networking on first boot.
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("bazboo"),
						Clevis: types.Clevis{
							Tang: []types.Tang{
								{
									Thumbprint: util.StrToPtr("mythumbprint"),
									URL:        "http://tang.example.com",
								},
							},
						},
					},
				},
			},
		},
		// Source with URL needs Net
		{
			Ignition: types.Ignition{
				Version: "3.3.0",
				Config: types.IgnitionConfig{
					Replace: types.Resource{
						Source: util.StrToPtr("http://example.com/config.ign"),
					},
				},
			},
		},
		// CustomClevis with NeedsNetwork set to true
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("foo"),
						Clevis: types.Clevis{
							Custom: types.ClevisCustom{
								NeedsNetwork: util.BoolToPtr(true),
							},
						},
					},
				},
			},
		},
		// Tang with adv explicitly set to nil needs networking on first boot.
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("foo"),
						Clevis: types.Clevis{
							Tang: []types.Tang{
								{
									Thumbprint:    util.StrToPtr("mythumbprint"),
									URL:           "http://mytang.example.com",
									Advertisement: nil,
								},
							},
						},
					},
				},
			},
		},
		// Multiple Tangs; one with adv set, one without needs networking on first boot.
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("foo"),
						Clevis: types.Clevis{
							Tang: []types.Tang{
								{
									Thumbprint:    util.StrToPtr("mythumbprint"),
									URL:           "http://mytang.example.com",
									Advertisement: util.StrToPtr(" {\"payload\": \"...\",\"protected\":\"...\",\"signature\":\"...\"}"),
								},
								{
									Thumbprint:    util.StrToPtr("mythumbprint"),
									URL:           "http://mytang.example.com",
									Advertisement: nil,
								},
							},
						},
					},
				},
			},
		},
		// Multiple Tangs with no adv set needs networking on first boot.
		{
			Storage: types.Storage{
				Luks: []types.Luks{
					{
						Name:   "foobar",
						Device: util.StrToPtr("foo"),
						Clevis: types.Clevis{
							Tang: []types.Tang{
								{
									Thumbprint:    util.StrToPtr("mythumbprint"),
									URL:           "http://mytang.example.com",
									Advertisement: util.StrToPtr(""),
								},
								{
									Thumbprint:    util.StrToPtr("mythumbprint"),
									URL:           "http://mytang.example.com",
									Advertisement: nil,
								},
							},
						},
					},
				},
			},
		},
	}

	for i, test := range tests {
		assert.Equal(t, checkNeedsNet(t, &test), true, "#%d: unexpected config doesn't need net: %v", i, test)
	}

}
