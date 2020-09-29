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

// The storage stage is responsible for partitioning disks, creating RAID
// arrays, formatting partitions, writing files, writing systemd units, and
// writing network units.

package fetch_offline

import (
	"net/url"
	"reflect"

	cfgutil "github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_3_experimental/types"
	"github.com/coreos/ignition/v2/internal/exec"
	"github.com/coreos/ignition/v2/internal/exec/stages"
	executil "github.com/coreos/ignition/v2/internal/exec/util"
	"github.com/coreos/ignition/v2/internal/log"
	"github.com/coreos/ignition/v2/internal/resource"
	"github.com/coreos/ignition/v2/internal/util"
)

const (
	name = "fetch-offline"
)

func init() {
	stages.Register(creator{})
}

type creator struct{}

func (creator) Create(logger *log.Logger, root string, _ resource.Fetcher) stages.Stage {
	return &stage{
		Util: executil.Util{
			DestDir: root,
			Logger:  logger,
		},
	}
}

func (creator) Name() string {
	return name
}

type stage struct {
	executil.Util
}

func (stage) Name() string {
	return name
}

func (s stage) Run(cfg types.Config) error {
	if needsNet, err := configNeedsNet(&cfg); err != nil {
		return err
	} else if needsNet {
		return exec.SignalNeedNet()
	}
	return nil
}

func configNeedsNet(cfg *types.Config) (bool, error) {
	return configNeedsNetRecurse(reflect.ValueOf(cfg))
}

func configNeedsNetRecurse(v reflect.Value) (bool, error) {
	t := v.Type()
	k := t.Kind()
	switch {
	case cfgutil.IsPrimitive(k):
		return false, nil
	case t == reflect.TypeOf(types.Resource{}):
		return sourceNeedsNet(v.Interface().(types.Resource))
	case t == reflect.TypeOf(types.Tang{}):
		return true, nil
	case k == reflect.Struct:
		for i := 0; i < v.NumField(); i += 1 {
			if needsNet, err := configNeedsNetRecurse(v.Field(i)); err != nil {
				return false, err
			} else if needsNet {
				return true, nil
			}
		}
	case k == reflect.Slice:
		for i := 0; i < v.Len(); i += 1 {
			if needsNet, err := configNeedsNetRecurse(v.Index(i)); err != nil {
				return false, err
			} else if needsNet {
				return true, nil
			}
		}
	case k == reflect.Ptr:
		v = v.Elem()
		if v.IsValid() {
			return configNeedsNetRecurse(v)
		}
	default:
		panic("unreachable code reached")
	}

	return false, nil
}

func sourceNeedsNet(res types.Resource) (bool, error) {
	if res.Source == nil {
		return false, nil
	}
	if u, err := url.Parse(*res.Source); err != nil {
		return false, err
	} else {
		return util.UrlNeedsNet(*u), nil
	}
}
