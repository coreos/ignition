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

#ifndef _BLKID_H_
#define _BLKID_H_

#include <string.h>
#include <stdbool.h>

typedef enum {
	RESULT_OK,
	RESULT_OPEN_FAILED,
	RESULT_PROBE_AMBIVALENT,
	RESULT_PROBE_FAILED,
	RESULT_LOOKUP_FAILED,
	RESULT_NO_PARTITION_TABLE,
	RESULT_BAD_INDEX,
	RESULT_GET_PARTLIST_FAILED,
	RESULT_GET_CACHE_FAILED,
	RESULT_DISK_HAS_NO_TYPE,
	RESULT_DISK_NOT_GPT,
	RESULT_BAD_PARAMS,
	RESULT_OVERFLOW,
	RESULT_MAX_BLOCK_DEVICES,
	RESULT_NO_TOPO,
	RESULT_NO_SECTOR_SIZE,
	RESULT_BAD_SECTOR_SIZE,
} result_t;

// really this shouldn't need to be larger than 145, but extra doesn't hurt
#define PART_INFO_BUF_SIZE 256

struct partition_info {
	char label[PART_INFO_BUF_SIZE];
	char uuid[PART_INFO_BUF_SIZE];
	char type_guid[PART_INFO_BUF_SIZE];
	long long start; // needs to be 64 bit
	long long size;  // to handle large partitions
	int number;
};

#define MAX_BLOCK_DEVICES 10
#define MAX_BLOCK_DEVICE_PATH_LEN 50

struct block_device_list {
	char path[MAX_BLOCK_DEVICES][MAX_BLOCK_DEVICE_PATH_LEN];
	int count;
};

result_t blkid_lookup(const char *device, bool allow_ambivalent, const char *field_name, char buf[], size_t buf_len);

result_t blkid_get_num_partitions(const char *device, int *ret);

result_t blkid_get_logical_sector_size(const char *device, int *ret_sector_size);

// WARNING part_num may not be what you expect. see the .c file's comment for why
result_t blkid_get_partition(const char *device, int part_num, struct partition_info *info);

result_t blkid_get_block_devices(const char *fstype, struct block_device_list *device);

#endif // _BLKID_H_
