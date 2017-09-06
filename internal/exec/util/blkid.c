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

static result_t extract_part_info(blkid_partition part, struct partition_info *info);

static char *safe_strncpy(char *dst, const char *src, size_t len);

// not quite strlcpy since it doesn't return the number of bytes copied,
// but still protects against missing the nul terminator
static char *safe_strncpy(char *dst, const char *src, size_t len) {
	char *ret = strncpy(dst, src, len);
	if (len > 0)
		dst[len-1] = '\0';

	return ret;
}

result_t blkid_lookup(const char *device, const char *field_name, char buf[], size_t buf_len)
{
	const char *field_val = "\0";

	blkid_probe pr = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;

	if (blkid_do_probe(pr) != 0)
		return RESULT_PROBE_FAILED;

	if (blkid_probe_has_value(pr, field_name))
		if (blkid_probe_lookup_value(pr, field_name, &field_val, NULL))
			return RESULT_LOOKUP_FAILED;

	safe_strncpy(buf, field_val, buf_len);

	blkid_free_probe(pr);

	return RESULT_OK;
}

// blkid_get_partition_list returns the partition list and the probe used to get it (so
// we can free it). It also performs a sanity check and ensures its is GPT formatted. In the case
// of empty partition tables it will return RESULT_GET_PARTLIST_FAILED. Ownership of the probe is
// passed on to the caller if RESULT_OK is returned, otherwise it frees the probe.
static result_t blkid_get_partition_list(const char *device, blkid_partlist *list_ret, blkid_probe *probe_ret)
{
	if (!device || !list_ret || !probe_ret)
		return RESULT_BAD_PARAMS;

	blkid_probe pr = blkid_new_probe_from_filename(device);
	if (!pr)
		return RESULT_OPEN_FAILED;

	blkid_partlist list = blkid_probe_get_partitions(pr);
	if (!list) {
		// This will be true if getting the partitions fails OR
		// there are no partitions.
		blkid_free_probe(pr);
		return RESULT_GET_PARTLIST_FAILED;
	}

	blkid_parttable table = blkid_partlist_get_table(list);
	if (!table) {
		blkid_free_probe(pr);
		return RESULT_NO_PARTITION_TABLE;
	}

	// sanity check, make sure we're not reading a MBR or something
	const char *str_type = blkid_parttable_get_type(table);
	if (!str_type) {
		blkid_free_probe(pr);
		return RESULT_DISK_HAS_NO_TYPE;
	}

	// unfortunately there doesn't seem to be a better check
	if (strncmp("gpt", str_type, sizeof("gpt")) != 0) {
		blkid_free_probe(pr);
		return RESULT_DISK_NOT_GPT;
	}

	*list_ret = list;
	*probe_ret = pr;
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
	blkid_probe pr = NULL;

	result_t err = blkid_get_partition_list(device, &list, &pr);
	if (err == RESULT_GET_PARTLIST_FAILED) {
		*n_parts_ret = 0;
		blkid_free_probe(pr);
		return RESULT_OK;
	} else if (err != RESULT_OK) {
		blkid_free_probe(pr);
		return err;
	}

	int tmp = blkid_partlist_numof_partitions(list);
	if (tmp == -1) {
		blkid_free_probe(pr);
		return RESULT_LOOKUP_FAILED;
	}

	*n_parts_ret = tmp;
	blkid_free_probe(pr);
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
	blkid_probe pr = NULL;
	result_t err = blkid_get_partition_list(device, &list, &pr);
	if (err != RESULT_OK) {
		// dont need to free the probe if it failed
		return err;
	}

	blkid_partition part = blkid_partlist_get_partition(list, part_num);
	if (!part) {
		blkid_free_probe(pr);
		return RESULT_BAD_INDEX;
	}

	err = extract_part_info(part, info);
	blkid_free_probe(pr);
	return err;
}

static result_t extract_part_info(blkid_partition part, struct partition_info *info)
{
	if (!part || !info)
		return RESULT_BAD_PARAMS;

	const char *ctmp = NULL; // doesn't need to be freed, will only point to memory owned by the probe
	long long itmp = 0;

	// label
	ctmp = blkid_partition_get_name(part);
	if (!ctmp)
		return RESULT_LOOKUP_FAILED;
	safe_strncpy(info->label, ctmp, PART_INFO_BUF_SIZE);

	// uuid
	ctmp = blkid_partition_get_uuid(part);
	if (!ctmp)
		return RESULT_LOOKUP_FAILED;
	safe_strncpy(info->uuid, ctmp, PART_INFO_BUF_SIZE);

	// type guid
	ctmp = blkid_partition_get_type_string(part);
	if (!ctmp)
		return RESULT_LOOKUP_FAILED;
	safe_strncpy(info->type_guid, ctmp, PART_INFO_BUF_SIZE);

	// part number
	itmp = blkid_partition_get_partno(part);
	if (itmp == -1)
		return RESULT_LOOKUP_FAILED;
	info->number = itmp;

	// start (in 512 byte sectors)
	// DANGER: check this works with nested partitions, see warning in libblkid docs
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
