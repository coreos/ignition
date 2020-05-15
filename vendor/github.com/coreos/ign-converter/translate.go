// Copyright 2019 Red Hat, Inc.
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

package ignconverter

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	old "github.com/coreos/ignition/config/v2_5_experimental/types"
	oldValidate "github.com/coreos/ignition/config/validate"
	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/config/validate"
)

// Error definitions

// NoFilesystemError type for when a filesystem is referenced in a config but there's no mapping to where
// it should be mounted (i.e. `path` in v3+ configs)
type NoFilesystemError string

func (e NoFilesystemError) Error() string {
	return fmt.Sprintf("Config defined filesystem %q but no mapping was defined."+
		"Please specify a path to be used as the filesystem mountpoint.", string(e))
}

// DuplicateInodeError is for when files, directories, or links both specify the same path
type DuplicateInodeError struct {
	Old string // first occurance of the path
	New string // second occurance of the path
}

func (e DuplicateInodeError) Error() string {
	return fmt.Sprintf("Config has conflicting inodes: %q and %q.  All files, directories and links must specify a unique `path`.", e.Old, e.New)
}

// UsesOwnLinkError is for when files, directories, or links use symlinks defined in the config
// in their own path. This is disallowed in v3+ configs.
type UsesOwnLinkError struct {
	LinkPath string
	Name     string
}

func (e UsesOwnLinkError) Error() string {
	return fmt.Sprintf("%s uses link defined in config %q. Please use a link not defined in Storage:Links", e.Name, e.LinkPath)
}

// ErrUsesNetworkd is the error for including networkd configs
var ErrUsesNetworkd = errors.New("config includes deprecated networkd section - use Files instead")

// Check returns if the config is translatable but does not do any translation.
// fsMap is a map from v2 filesystem names to the paths under which they should
// be mounted in v3.
func Check(cfg old.Config, fsMap map[string]string) error {
	rpt := oldValidate.ValidateWithoutSource(reflect.ValueOf(cfg))
	if rpt.IsFatal() || rpt.IsDeprecated() {
		// disallow any deprecated fields
		return fmt.Errorf("Invalid input config:\n%s", rpt.String())
	}

	if len(cfg.Networkd.Units) != 0 {
		return ErrUsesNetworkd
	}

	// check that all filesystems have a path
	if fsMap == nil {
		fsMap = map[string]string{}
	}
	fsMap["root"] = "/"
	for _, fs := range cfg.Storage.Filesystems {
		if _, ok := fsMap[fs.Name]; !ok {
			return NoFilesystemError(fs.Name)
		}
	}

	// check that there are no duplicates with files, links, or directories
	// from path to a pretty-printing description of the entry
	entryMap := map[string]string{}
	links := make([]string, 0, len(cfg.Storage.Links))
	// build up a list of all the links we write. We're not allow to use links
	// that we write
	for _, link := range cfg.Storage.Links {
		path := filepath.Join("/", fsMap[link.Filesystem], link.Path)
		links = append(links, path)
	}

	for _, file := range cfg.Storage.Files {
		path := filepath.Join("/", fsMap[file.Filesystem], file.Path)
		name := fmt.Sprintf("File: %s", path)
		if duplicate, isDup := entryMap[path]; isDup {
			return DuplicateInodeError{duplicate, name}
		}
		if l := checkPathUsesLink(links, path); l != "" {
			return &UsesOwnLinkError{
				LinkPath: l,
				Name:     name,
			}
		}
		entryMap[path] = name
	}
	for _, dir := range cfg.Storage.Directories {
		path := filepath.Join("/", fsMap[dir.Filesystem], dir.Path)
		name := fmt.Sprintf("Directory: %s", path)
		if duplicate, isDup := entryMap[path]; isDup {
			return DuplicateInodeError{duplicate, name}
		}
		if l := checkPathUsesLink(links, path); l != "" {
			return &UsesOwnLinkError{
				LinkPath: l,
				Name:     name,
			}
		}
		entryMap[path] = name
	}
	for _, link := range cfg.Storage.Links {
		path := filepath.Join("/", fsMap[link.Filesystem], link.Path)
		name := fmt.Sprintf("Link: %s", path)
		if duplicate, isDup := entryMap[path]; isDup {
			return &DuplicateInodeError{duplicate, name}
		}
		entryMap[path] = name
		if l := checkPathUsesLink(links, path); l != "" {
			return &UsesOwnLinkError{
				LinkPath: l,
				Name:     name,
			}
		}
	}
	return nil
}

func checkPathUsesLink(links []string, path string) string {
	for _, l := range links {
		if strings.HasPrefix(path, l) && path != l {
			return l
		}
	}
	return ""
}

// Translate translates an Ignition spec v2 config to v3
func Translate(cfg old.Config, fsMap map[string]string) (types.Config, error) {
	if err := Check(cfg, fsMap); err != nil {
		return types.Config{}, err
	}
	res := types.Config{
		// Ignition section
		Ignition: types.Ignition{
			Version: "3.2.0-experimental",
			Config: types.IgnitionConfig{
				Replace: translateCfgRef(cfg.Ignition.Config.Replace),
				Merge:   translateCfgRefs(cfg.Ignition.Config.Append),
			},
			Security: types.Security{
				TLS: types.TLS{
					CertificateAuthorities: translateCAs(cfg.Ignition.Security.TLS.CertificateAuthorities),
				},
			},
			Timeouts: types.Timeouts{
				HTTPResponseHeaders: cfg.Ignition.Timeouts.HTTPResponseHeaders,
				HTTPTotal:           cfg.Ignition.Timeouts.HTTPTotal,
			},
		},
		// Passwd section
		Passwd: types.Passwd{
			Users:  translateUsers(cfg.Passwd.Users),
			Groups: translateGroups(cfg.Passwd.Groups),
		},
		Systemd: types.Systemd{
			Units: translateUnits(cfg.Systemd.Units),
		},
		Storage: types.Storage{
			Disks:       translateDisks(cfg.Storage.Disks),
			Raid:        translateRaid(cfg.Storage.Raid),
			Filesystems: translateFilesystems(cfg.Storage.Filesystems, fsMap),
			Files:       translateFiles(cfg.Storage.Files, fsMap),
			Directories: translateDirectories(cfg.Storage.Directories, fsMap),
			Links:       translateLinks(cfg.Storage.Links, fsMap),
		},
	}
	r := validate.ValidateWithContext(res, nil)
	if r.IsFatal() {
		return types.Config{}, errors.New(r.String())
	}
	return res, nil
}

func translateCfgRef(ref *old.ConfigReference) (ret types.Resource) {
	if ref == nil {
		return
	}
	ret.Source = &ref.Source
	ret.Verification.Hash = ref.Verification.Hash
	return
}

func translateCfgRefs(refs []old.ConfigReference) (ret []types.Resource) {
	for _, ref := range refs {
		ret = append(ret, translateCfgRef(&ref))
	}
	return
}

func translateCAs(refs []old.CaReference) (ret []types.Resource) {
	for _, ref := range refs {
		ret = append(ret, types.Resource{
			Source: &ref.Source,
			Verification: types.Verification{
				Hash: ref.Verification.Hash,
			},
		})
	}
	return
}

func translateUsers(users []old.PasswdUser) (ret []types.PasswdUser) {
	for _, u := range users {
		ret = append(ret, types.PasswdUser{
			Name:              u.Name,
			PasswordHash:      u.PasswordHash,
			SSHAuthorizedKeys: translateUserSSH(u.SSHAuthorizedKeys),
			UID:               u.UID,
			Gecos:             strP(u.Gecos),
			HomeDir:           strP(u.HomeDir),
			NoCreateHome:      boolP(u.NoCreateHome),
			PrimaryGroup:      strP(u.PrimaryGroup),
			Groups:            translateUserGroups(u.Groups),
			NoUserGroup:       boolP(u.NoUserGroup),
			NoLogInit:         boolP(u.NoLogInit),
			Shell:             strP(u.Shell),
			System:            boolP(u.System),
		})
	}
	return
}

func translateUserSSH(in []old.SSHAuthorizedKey) (ret []types.SSHAuthorizedKey) {
	for _, k := range in {
		ret = append(ret, types.SSHAuthorizedKey(k))
	}
	return
}

func translateUserGroups(in []old.Group) (ret []types.Group) {
	for _, g := range in {
		ret = append(ret, types.Group(g))
	}
	return
}

func translateGroups(groups []old.PasswdGroup) (ret []types.PasswdGroup) {
	for _, g := range groups {
		ret = append(ret, types.PasswdGroup{
			Name:         g.Name,
			Gid:          g.Gid,
			PasswordHash: strP(g.PasswordHash),
			System:       boolP(g.System),
		})
	}
	return
}

func translateUnits(units []old.Unit) (ret []types.Unit) {
	for _, u := range units {
		ret = append(ret, types.Unit{
			Name:     u.Name,
			Enabled:  u.Enabled,
			Mask:     boolP(u.Mask),
			Contents: strP(u.Contents),
			Dropins:  translateDropins(u.Dropins),
		})
	}
	return
}

func translateDropins(dropins []old.SystemdDropin) (ret []types.Dropin) {
	for _, d := range dropins {
		ret = append(ret, types.Dropin{
			Name:     d.Name,
			Contents: strP(d.Contents),
		})
	}
	return
}

func translateDisks(disks []old.Disk) (ret []types.Disk) {
	for _, d := range disks {
		ret = append(ret, types.Disk{
			Device:     d.Device,
			WipeTable:  boolP(d.WipeTable),
			Partitions: translatePartitions(d.Partitions),
		})
	}
	return
}

func translatePartitions(parts []old.Partition) (ret []types.Partition) {
	for _, p := range parts {
		ret = append(ret, types.Partition{
			Label:              p.Label,
			Number:             p.Number,
			SizeMiB:            p.SizeMiB,
			StartMiB:           p.StartMiB,
			TypeGUID:           strP(p.TypeGUID),
			GUID:               strP(p.GUID),
			WipePartitionEntry: boolP(p.WipePartitionEntry),
			ShouldExist:        p.ShouldExist,
		})
	}
	return
}

func translateRaid(raids []old.Raid) (ret []types.Raid) {
	for _, r := range raids {
		ret = append(ret, types.Raid{
			Name:    r.Name,
			Level:   r.Level,
			Devices: translateDevices(r.Devices),
			Spares:  intP(r.Spares),
			Options: translateRaidOptions(r.Options),
		})
	}
	return
}

func translateDevices(devices []old.Device) (ret []types.Device) {
	for _, d := range devices {
		ret = append(ret, types.Device(d))
	}
	return
}

func translateRaidOptions(options []old.RaidOption) (ret []types.RaidOption) {
	for _, o := range options {
		ret = append(ret, types.RaidOption(o))
	}
	return
}

func translateFilesystems(fss []old.Filesystem, m map[string]string) (ret []types.Filesystem) {
	for _, f := range fss {
		if f.Name == "root" {
			// root is implied
			continue
		}
		if f.Mount == nil {
			f.Mount = &old.Mount{}
		}
		ret = append(ret, types.Filesystem{
			Device:         f.Mount.Device,
			Format:         strP(f.Mount.Format),
			WipeFilesystem: boolP(f.Mount.WipeFilesystem),
			Label:          f.Mount.Label,
			UUID:           f.Mount.UUID,
			Options:        translateFilesystemOptions(f.Mount.Options),
			Path:           strP(m[f.Name]),
		})
	}
	return
}

func translateFilesystemOptions(options []old.MountOption) (ret []types.FilesystemOption) {
	for _, o := range options {
		ret = append(ret, types.FilesystemOption(o))
	}
	return
}

func translateNode(n old.Node, m map[string]string) types.Node {
	if n.User == nil {
		n.User = &old.NodeUser{}
	}
	if n.Group == nil {
		n.Group = &old.NodeGroup{}
	}
	return types.Node{
		Path: filepath.Join(m[n.Filesystem], n.Path),
		User: types.NodeUser{
			ID:   n.User.ID,
			Name: strP(n.User.Name),
		},
		Group: types.NodeGroup{
			ID:   n.Group.ID,
			Name: strP(n.Group.Name),
		},
		Overwrite: n.Overwrite,
	}
}

func translateFiles(files []old.File, m map[string]string) (ret []types.File) {
	for _, f := range files {
		// 2.x files are overwrite by default
		if f.Node.Overwrite == nil {
			f.Node.Overwrite = boolP(true)
		}
		file := types.File{
			Node: translateNode(f.Node, m),
			FileEmbedded1: types.FileEmbedded1{
				Mode: f.Mode,
			},
		}
		c := types.Resource{
			Compression: strP(f.Contents.Compression),
			Source:      strPStrict(f.Contents.Source),
		}
		c.Verification.Hash = f.FileEmbedded1.Contents.Verification.Hash

		if f.Append {
			file.Append = []types.Resource{c}
		} else {
			file.Contents = c
		}
		ret = append(ret, file)
	}
	return
}

func translateLinks(links []old.Link, m map[string]string) (ret []types.Link) {
	for _, l := range links {
		ret = append(ret, types.Link{
			Node: translateNode(l.Node, m),
			LinkEmbedded1: types.LinkEmbedded1{
				Hard:   boolP(l.Hard),
				Target: l.Target,
			},
		})
	}
	return
}

func translateDirectories(dirs []old.Directory, m map[string]string) (ret []types.Directory) {
	for _, d := range dirs {
		ret = append(ret, types.Directory{
			Node: translateNode(d.Node, m),
			DirectoryEmbedded1: types.DirectoryEmbedded1{
				Mode: d.Mode,
			},
		})
	}
	return
}
