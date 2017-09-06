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

static result_t extract_part_info(blkid_partition part, struct partition_info *info);

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

result_t blkid_lookup(const char *device, const char *field_name, char *buf, size_t buf_len)
{
	const char *field_val = "\0";

	blkid_probe pr _cleanup_probe_ = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;

	if (blkid_do_probe(pr) < 0)
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

	return extract_part_info(part, info);
}

static result_t extract_part_info(blkid_partition part, struct partition_info *info)
{
	if (!part || !info)
		return RESULT_BAD_PARAMS;

	const char *ctmp = NULL;
	long long itmp = 0;
	int err;

	// the blkid probe owns the memory returned by blkid_get_* and will free it with the probe.

	// label
	ctmp = blkid_partition_get_name(part);
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

	// start (in 512 byte sectors)
	itmp = blkid_partition_get_start(part);
	if (itmp == -1)
		return RESULT_LOOKUP_FAILED;
	info->start = itmp;

	// size (in 512 byte sectors)
	itmp = blkid_partition_get_size(part);
	if (itmp == -1)
		return RESULT_LOOKUP_FAILED;
	info->size = itmp;

	return RESULT_OK;
}
