// Copyright 2016 CoreOS, Inc.
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

#include <blkid/blkid.h>
#include "blkid.h"

// blkid_free_probe is safe to call with NULL pointers
static inline void _free_probe(blkid_probe *pr) { if (pr) blkid_free_probe(*pr); }
#define _cleanup_probe_ __attribute__((cleanup(_free_probe)))

static inline void _free_cache(blkid_cache *gcache) { blkid_put_cache(*gcache); }
#define _cleanup_cache_ __attribute__((cleanup(_free_cache)))

static inline void _free_iterator(blkid_dev_iterate *iter) { blkid_dev_iterate_end(*iter); }
#define _cleanup_iterator_ __attribute__((cleanup(_free_iterator)))

static result_t extract_part_info(blkid_partition part, struct partition_info *info, long sector_divisor);

static result_t checked_copy(char *dest, const char *src, size_t len);

// checked_copy is a helper for extract_part_info. It checks if src is null and fits in dst.
// It's basically a safer strncpy to reduce boilerplate
// Returns:
//   RESULT_LOOKUP_FAILED if src is null
//   RESULT_OVERFLOW if src is too big
//   RESULT_OK on success
static result_t checked_copy(char *dest, const char *src, size_t len) {
	if (!src)
		return RESULT_LOOKUP_FAILED;

	if (strlen(src) + 1 > len)
		return RESULT_OVERFLOW;

	strncpy(dest, src, len);
	return RESULT_OK;
}

result_t blkid_lookup(const char *device, bool allow_ambivalent, const char *field_name, char *buf, size_t buf_len)
{
	const char *field_val = "\0";
	int ret;

	blkid_probe pr _cleanup_probe_ = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;

	if (allow_ambivalent) {
		ret = blkid_do_probe(pr);
	} else {
		ret = blkid_do_safeprobe(pr);
		if (ret == -2)
			return RESULT_PROBE_AMBIVALENT;
	}
	if (ret < 0)
		return RESULT_PROBE_FAILED;

	if (blkid_probe_has_value(pr, field_name))
		if (blkid_probe_lookup_value(pr, field_name, &field_val, NULL))
			return RESULT_LOOKUP_FAILED;

	return checked_copy(buf, field_val, buf_len);
}

// blkid_get_partition_list returns the partition list for a given opened probe. It also performs
// checks it is GPT formatted. In the case of empty partition tables it will return
// RESULT_GET_PARTLIST_FAILED.
static result_t blkid_get_partition_list(blkid_probe pr, blkid_partlist *list_ret)
{
	if (!list_ret || !pr)
		return RESULT_BAD_PARAMS;

	blkid_partlist list = blkid_probe_get_partitions(pr);
	if (!list) {
		// This will be true if getting the partitions fails OR
		// there are no partitions.
		return RESULT_GET_PARTLIST_FAILED;
	}

	blkid_parttable table = blkid_partlist_get_table(list);
	if (!table)
		return RESULT_NO_PARTITION_TABLE;

	// sanity check, make sure we're not reading a MBR or something
	const char *str_type = blkid_parttable_get_type(table);
	if (!str_type)
		return RESULT_DISK_HAS_NO_TYPE;

	// unfortunately there doesn't seem to be a better check
	if (strncmp("gpt", str_type, sizeof("gpt")) != 0)
		return RESULT_DISK_NOT_GPT;

	*list_ret = list;
	return RESULT_OK;
}

// blkid_get_num_partitions sets *n_parts_ret to the number of partitions on device and
// returns a result_t indicating if it was successful. *n_parts_ret is not changed if
// RESULT_OK is not returned.
result_t blkid_get_num_partitions(const char *device, int *n_parts_ret)
{
	if (!device || !n_parts_ret)
		return RESULT_BAD_PARAMS;

	blkid_partlist list = NULL;
	blkid_probe pr _cleanup_probe_ = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;

	result_t err = blkid_get_partition_list(pr, &list);
	if (err == RESULT_GET_PARTLIST_FAILED) {
		*n_parts_ret = 0;
		return RESULT_OK;
	} else if (err != RESULT_OK) {
		return err;
	}

	int tmp = blkid_partlist_numof_partitions(list);
	if (tmp == -1)
		return RESULT_LOOKUP_FAILED;

	*n_parts_ret = tmp;
	return RESULT_OK;
}

result_t blkid_get_logical_sector_size(const char *device, int *ret_sector_size) {
	if (!device || !ret_sector_size)
		return RESULT_BAD_PARAMS;

	blkid_probe pr _cleanup_probe_ = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;
	
	// topo points inside of pr and will be freed when pr is freed
	blkid_topology topo = blkid_probe_get_topology(pr);
	if (!topo) {
		return RESULT_NO_TOPO;
	}

	long sector_size = blkid_topology_get_logical_sector_size(topo);
	if (sector_size == 0) {
		return RESULT_NO_SECTOR_SIZE;
	}
	if (sector_size % 512 != 0) {
		return RESULT_BAD_SECTOR_SIZE;
	}

	*ret_sector_size = sector_size;
	return RESULT_OK;
}

// WARNING, part_num is probably not what you expect!
// part_num refers to a number 0..blkid_get_num_partitions()-1, NOT
// the partition number like in /dev/sdaX. See blkid_partlist_devno_to_partition()'s
// documentation if you need the latter case.
result_t blkid_get_partition(const char *device, int part_num, struct partition_info *info)
{
	if (!info || !device || part_num < 0)
		return RESULT_BAD_PARAMS;

	blkid_partlist list = NULL;
	blkid_probe pr _cleanup_probe_ = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;

	result_t err = blkid_get_partition_list(pr, &list);
	if (err != RESULT_OK)
		return err;

	blkid_partition part = blkid_partlist_get_partition(list, part_num);
	if (!part)
		return RESULT_BAD_INDEX;

	// topo points inside of pr and will be freed when pr is freed
	blkid_topology topo = blkid_probe_get_topology(pr);
	if (!topo) {
		return RESULT_NO_TOPO;
	}

	long sector_size = blkid_topology_get_logical_sector_size(topo);
	if (sector_size == 0) {
		return RESULT_NO_SECTOR_SIZE;
	}
	if (sector_size % 512 != 0) {
		return RESULT_BAD_SECTOR_SIZE;
	}

	return extract_part_info(part, info, sector_size / 512);
}

// extract_part_info reads the information for a partition into *info. sector_divisor is how many 512
// byte sectors are in a logical sector (1 for "normal" sectors, 8 for 4k sectors). This is needed because
// libblkid always assumes 512 byte sectors regardless of what the actual logical sector size of the device is.
static result_t extract_part_info(blkid_partition part, struct partition_info *info, long sector_divisor)
{
	if (!part || !info)
		return RESULT_BAD_PARAMS;

	const char *ctmp = NULL;
	long long itmp = 0;
	int err;

	// the blkid probe owns the memory returned by blkid_get_* and will free it with the probe.

	// label
	ctmp = blkid_partition_get_name(part);
	// If the GPT label is empty, then libblkid will return NULL instead of an empty string.
	// There is no NULL value in GPT, so just reset to empty.
	if (!ctmp)
		ctmp = "";
	err = checked_copy(info->label, ctmp, PART_INFO_BUF_SIZE);
	if (err)
		return err;

	// uuid
	ctmp = blkid_partition_get_uuid(part);
	err = checked_copy(info->uuid, ctmp, PART_INFO_BUF_SIZE);
	if (err)
		return err;

	// type guid
	ctmp = blkid_partition_get_type_string(part);
	err = checked_copy(info->type_guid, ctmp, PART_INFO_BUF_SIZE);
	if (err)
		return err;

	// part number
	itmp = blkid_partition_get_partno(part);
	if (itmp == -1)
		return RESULT_LOOKUP_FAILED;
	info->number = itmp;

	// start (in sectors)
	itmp = blkid_partition_get_start(part);
	if (itmp == -1)
		return RESULT_LOOKUP_FAILED;
	info->start = itmp / sector_divisor;

	// size (in sectors)
	itmp = blkid_partition_get_size(part);
	if (itmp == -1)
		return RESULT_LOOKUP_FAILED;
	info->size = itmp / sector_divisor;

	return RESULT_OK;
}

// blkid_get_block_devices fetches block devices with the given FSTYPE
result_t blkid_get_block_devices(const char *fstype, struct block_device_list *device) {
	blkid_dev_iterate iter _cleanup_iterator_ = NULL;
	blkid_dev dev;
	blkid_cache cache _cleanup_cache_ = NULL;
	int err, count = 0;
	const char *ctmp;
	
	if (blkid_get_cache(&cache, "/dev/null") != 0)
		return RESULT_GET_CACHE_FAILED;
	
	if (blkid_probe_all(cache) != 0)
		return RESULT_PROBE_FAILED;

	iter = blkid_dev_iterate_begin(cache);
	
	blkid_dev_set_search(iter, "TYPE", fstype);
	
	while (blkid_dev_next(iter, &dev) == 0) {
		dev = blkid_verify(cache, dev);
		if (!dev)
			continue;
		if (count >= MAX_BLOCK_DEVICES)
			return RESULT_MAX_BLOCK_DEVICES;
		ctmp = blkid_dev_devname(dev);
		err = checked_copy(device->path[count], ctmp, MAX_BLOCK_DEVICE_PATH_LEN);
		if (err)
			return err;
		count++;
	}
	
	device->count = count;
	return RESULT_OK;
}
