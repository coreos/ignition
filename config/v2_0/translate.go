// Copyright 2018 CoreOS, Inc.
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

package v2_0

func TranslateFromV1(old v1.Config) types.Config {
	config := types.Config{
		Ignition: types.Ignition{
			Version: v2_0.MaxVersion.String(),
		},
	}

	for _, oldDisk := range old.Storage.Disks {
		disk := types.Disk{
			Device:    string(oldDisk.Device),
			WipeTable: oldDisk.WipeTable,
		}

		for _, oldPartition := range oldDisk.Partitions {
			disk.Partitions = append(disk.Partitions, types.Partition{
				Label:    string(oldPartition.Label),
				Number:   oldPartition.Number,
				Size:     int(oldPartition.Size),
				Start:    int(oldPartition.Start),
				TypeGUID: string(oldPartition.TypeGUID),
			})
		}

		config.Storage.Disks = append(config.Storage.Disks, disk)
	}

	for _, oldArray := range old.Storage.Arrays {
		array := types.Raid{
			Name:   oldArray.Name,
			Level:  oldArray.Level,
			Spares: oldArray.Spares,
		}

		for _, oldDevice := range oldArray.Devices {
			array.Devices = append(array.Devices, types.Device(oldDevice))
		}

		config.Storage.Raid = append(config.Storage.Raid, array)
	}

	for i, oldFilesystem := range old.Storage.Filesystems {
		filesystem := types.Filesystem{
			Name: fmt.Sprintf("_translate-filesystem-%d", i),
			Mount: &types.Mount{
				Device: string(oldFilesystem.Device),
				Format: string(oldFilesystem.Format),
			},
		}

		if oldFilesystem.Create != nil {
			filesystem.Mount.Create = &types.Create{
				Force:   oldFilesystem.Create.Force,
				Options: translateV1MkfsOptionsToOptionSlice(oldFilesystem.Create.Options),
			}
		}

		config.Storage.Filesystems = append(config.Storage.Filesystems, filesystem)

		for _, oldFile := range oldFilesystem.Files {
			file := types.File{
				Node: types.Node{
					Filesystem: filesystem.Name,
					Path:       string(oldFile.Path),
					User:       &types.NodeUser{ID: intToPtr(oldFile.Uid)},
					Group:      &types.NodeGroup{ID: intToPtr(oldFile.Gid)},
				},
				FileEmbedded1: types.FileEmbedded1{
					Mode: intToPtr(int(oldFile.Mode)),
					Contents: types.FileContents{
						Source: (&url.URL{
							Scheme: "data",
							Opaque: "," + dataurl.EscapeString(oldFile.Contents),
						}).String(),
					},
				},
			}

			config.Storage.Files = append(config.Storage.Files, file)
		}
	}

	for _, oldUnit := range old.Systemd.Units {
		unit := types.Unit{
			Name:     string(oldUnit.Name),
			Enable:   oldUnit.Enable,
			Mask:     oldUnit.Mask,
			Contents: oldUnit.Contents,
		}

		for _, oldDropIn := range oldUnit.DropIns {
			unit.Dropins = append(unit.Dropins, types.SystemdDropin{
				Name:     string(oldDropIn.Name),
				Contents: oldDropIn.Contents,
			})
		}

		config.Systemd.Units = append(config.Systemd.Units, unit)
	}

	for _, oldUnit := range old.Networkd.Units {
		config.Networkd.Units = append(config.Networkd.Units, types.Networkdunit{
			Name:     string(oldUnit.Name),
			Contents: oldUnit.Contents,
		})
	}

	for _, oldUser := range old.Passwd.Users {
		user := types.PasswdUser{
			Name:              oldUser.Name,
			PasswordHash:      strToPtr(oldUser.PasswordHash),
			SSHAuthorizedKeys: translateStringSliceToSSHAuthorizedKeySlice(oldUser.SSHAuthorizedKeys),
		}

		if oldUser.Create != nil {
			var uid *int
			if oldUser.Create.Uid != nil {
				tmp := int(*oldUser.Create.Uid)
				uid = &tmp
			}

			user.Create = &types.Usercreate{
				UID:          uid,
				Gecos:        oldUser.Create.GECOS,
				HomeDir:      oldUser.Create.Homedir,
				NoCreateHome: oldUser.Create.NoCreateHome,
				PrimaryGroup: oldUser.Create.PrimaryGroup,
				Groups:       translateStringSliceToUsercreateGroupSlice(oldUser.Create.Groups),
				NoUserGroup:  oldUser.Create.NoUserGroup,
				System:       oldUser.Create.System,
				NoLogInit:    oldUser.Create.NoLogInit,
				Shell:        oldUser.Create.Shell,
			}
		}

		config.Passwd.Users = append(config.Passwd.Users, user)
	}

	for _, oldGroup := range old.Passwd.Groups {
		var gid *int
		if oldGroup.Gid != nil {
			tmp := int(*oldGroup.Gid)
			gid = &tmp
		}
		config.Passwd.Groups = append(config.Passwd.Groups, types.PasswdGroup{
			Name:         oldGroup.Name,
			Gid:          gid,
			PasswordHash: oldGroup.PasswordHash,
			System:       oldGroup.System,
		})
	}

	return config
}

// golang--
func translateV1MkfsOptionsToOptionSlice(opts v1.MkfsOptions) []types.CreateOption {
	newOpts := make([]types.CreateOption, len(opts))
	for i, o := range opts {
		newOpts[i] = types.CreateOption(o)
	}
	return newOpts
}

// golang--
func translateStringSliceToSSHAuthorizedKeySlice(keys []string) []types.SSHAuthorizedKey {
	newKeys := make([]types.SSHAuthorizedKey, len(keys))
	for i, k := range keys {
		newKeys[i] = types.SSHAuthorizedKey(k)
	}
	return newKeys
}

// golang--
func translateStringSliceToUsercreateGroupSlice(groups []string) []types.UsercreateGroup {
	var newGroups []types.UsercreateGroup
	for _, g := range groups {
		newGroups = append(newGroups, types.UsercreateGroup(g))
	}
	return newGroups
}
