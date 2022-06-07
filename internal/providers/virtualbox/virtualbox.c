// Copyright 2021 Red Hat, Inc.
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

#include <linux/vboxguest.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include "virtualbox.h"

// From virtualbox/include/VBox/HostServices/GuestPropertySvc.h
#define GUEST_PROP_FN_GET_PROP 1
#define GUEST_PROP_FN_DEL_PROP 4

static void _cleanup_close(int *fd) {
	if (*fd != -1) {
		close(*fd);
	}
}
#define _cleanup_close_ __attribute__((cleanup(_cleanup_close)))

static void _cleanup_free(void *ptr) {
	free(*(void **)ptr);
}
#define _cleanup_free_ __attribute__((cleanup(_cleanup_free)))

static void init_header(struct vbg_ioctl_hdr *hdr, size_t size_in, size_t size_out) {
	hdr->size_in = sizeof(*hdr) + size_in;
	hdr->version = VBG_IOCTL_HDR_VERSION;
	hdr->type = VBG_IOCTL_HDR_TYPE_DEFAULT;
	hdr->size_out = sizeof(*hdr) + size_out;
}

static int version_info(int fd) {
	struct vbg_ioctl_driver_version_info msg = {
		.u = {
			.in = {
				.req_version = 0x00010000,
				.min_version = 0x00010000,
			},
		},
	};
	init_header(&msg.hdr, sizeof(msg.u.in), sizeof(msg.u.out));
	if (ioctl(fd, VBG_IOCTL_DRIVER_VERSION_INFO, &msg)) {
		return VERR_GENERAL_FAILURE;
	}
	return msg.hdr.rc;
}

static int connect(int fd, uint32_t *client_id) {
	struct vbg_ioctl_hgcm_connect msg = {
		.u = {
			.in = {
				.loc = {
					.type = VMMDEV_HGCM_LOC_LOCALHOST_EXISTING,
					.u = {
						.localhost = {
							.service_name = "VBoxGuestPropSvc",
						},
					},
				},
			},
		},
	};
	init_header(&msg.hdr, sizeof(msg.u.in), sizeof(msg.u.out));
	if (ioctl(fd, VBG_IOCTL_HGCM_CONNECT, &msg)) {
		return VERR_GENERAL_FAILURE;
	}
	if (msg.hdr.rc != VINF_SUCCESS) {
		return msg.hdr.rc;
	}
	*client_id = msg.u.out.client_id;
	return VINF_SUCCESS;
}

static int get_prop(int fd, uint32_t client_id, const char *name, void **value, size_t *size) {
	// xref VbglR3GuestPropRead() in
	// virtualbox/src/VBox/Additions/common/VBoxGuest/lib/VBoxGuestR3LibGuestProp.cpp

	// init header
	size_t msg_size = sizeof(struct vbg_ioctl_hgcm_call) + 4 * sizeof(struct vmmdev_hgcm_function_parameter64);
	struct vbg_ioctl_hgcm_call _cleanup_free_ *msg = calloc(1, msg_size);
	if (msg == NULL) {
		return VERR_NO_MEMORY;
	}
	// init_header re-adds the size of msg->hdr
	init_header(&msg->hdr, msg_size - sizeof(msg->hdr), msg_size - sizeof(msg->hdr));
	msg->client_id = client_id;
	msg->function = GUEST_PROP_FN_GET_PROP;
	msg->timeout_ms = -1;  // inf
	msg->interruptible = 1;
	msg->parm_count = 4;

	// init arguments
	char ch;
	struct vmmdev_hgcm_function_parameter64 *params = (void *) (msg + 1);
	// property name (in)
	params[0].type = VMMDEV_HGCM_PARM_TYPE_LINADDR_IN;
	params[0].u.pointer.size = strlen(name) + 1;
	params[0].u.pointer.u.linear_addr = (uintptr_t) name;
	// property value (out)
	params[1].type = VMMDEV_HGCM_PARM_TYPE_LINADDR;
	params[1].u.pointer.size = 1;
	params[1].u.pointer.u.linear_addr = (uintptr_t) &ch;
	// property timestamp (out)
	params[2].type = VMMDEV_HGCM_PARM_TYPE_64BIT;
	// property size (out)
	params[3].type = VMMDEV_HGCM_PARM_TYPE_32BIT;

	// get value size
	if (ioctl(fd, VBG_IOCTL_HGCM_CALL_64(msg_size), msg)) {
		return VERR_GENERAL_FAILURE;
	}
	switch (msg->hdr.rc) {
	case VINF_SUCCESS:
	case VERR_BUFFER_OVERFLOW:
		// allocate buffer
		; // labels can't point to declarations
		size_t buf_size = params[3].u.value32;
		void _cleanup_free_ *buf = malloc(buf_size);
		if (buf == NULL) {
			return VERR_NO_MEMORY;
		}
		params[1].u.pointer.size = buf_size;
		params[1].u.pointer.u.linear_addr = (uintptr_t) buf;

		// get value
		if (ioctl(fd, VBG_IOCTL_HGCM_CALL_64(msg_size), msg)) {
			return VERR_GENERAL_FAILURE;
		}
		if (msg->hdr.rc != VINF_SUCCESS) {
			return msg->hdr.rc;
		}
		*value = buf;
		buf = NULL;
		*size = buf_size;
		return VINF_SUCCESS;
	case VERR_NOT_FOUND:
		*value = NULL;
		*size = 0;
		return VINF_SUCCESS;
	default:
		return msg->hdr.rc;
	}
}

static int del_prop(int fd, uint32_t client_id, const char *name) {
	// xref VbglR3GuestPropDelete() in
	// virtualbox/src/VBox/Additions/common/VBoxGuest/lib/VBoxGuestR3LibGuestProp.cpp

	// init header
	size_t msg_size = sizeof(struct vbg_ioctl_hgcm_call) + sizeof(struct vmmdev_hgcm_function_parameter64);
	struct vbg_ioctl_hgcm_call _cleanup_free_ *msg = calloc(1, msg_size);
	if (msg == NULL) {
		return VERR_NO_MEMORY;
	}
	// init_header re-adds the size of msg->hdr
	init_header(&msg->hdr, msg_size - sizeof(msg->hdr), msg_size - sizeof(msg->hdr));
	msg->client_id = client_id;
	msg->function = GUEST_PROP_FN_DEL_PROP;
	msg->timeout_ms = -1;  // inf
	msg->interruptible = 1;
	msg->parm_count = 1;

	// init arguments
	struct vmmdev_hgcm_function_parameter64 *params = (void *) (msg + 1);
	// property name (in)
	params[0].type = VMMDEV_HGCM_PARM_TYPE_LINADDR_IN;
	params[0].u.pointer.size = strlen(name) + 1;
	params[0].u.pointer.u.linear_addr = (uintptr_t) name;

	// delete value
	if (ioctl(fd, VBG_IOCTL_HGCM_CALL_64(msg_size), msg)) {
		return VERR_GENERAL_FAILURE;
	}
	if (msg->hdr.rc != VINF_SUCCESS) {
		return msg->hdr.rc;
	}
	return VINF_SUCCESS;
}

static int disconnect(int fd, uint32_t client_id) {
	struct vbg_ioctl_hgcm_disconnect msg = {
		.u = {
			.in = {
				.client_id = client_id,
			},
		},
	};
	init_header(&msg.hdr, sizeof(msg.u.in), 0);
	if (ioctl(fd, VBG_IOCTL_HGCM_DISCONNECT, &msg)) {
		return VERR_GENERAL_FAILURE;
	}
	return msg.hdr.rc;
}

static int start_connection(uint32_t *client_id) {
	// clear any previous garbage in errno for error returns
	errno = 0;

	// open character device
	int _cleanup_close_ fd = open("/dev/vboxguest", O_RDWR | O_CLOEXEC);
	if (fd == -1) {
		return VERR_GENERAL_FAILURE;
	}

	// negotiate protocol version
	int ret = version_info(fd);
	if (ret != VINF_SUCCESS) {
		return ret;
	}

	// connect to property service
	ret = connect(fd, client_id);
	if (ret != VINF_SUCCESS) {
		return ret;
	}

	// return fd
	ret = fd;
	fd = -1;
	return ret;
}

int virtualbox_get_guest_property(char *name, void **value, size_t *size) {
	// connect
	uint32_t client_id;
	int ret = start_connection(&client_id);
	if (ret < 0) {
		return ret;
	}
	int _cleanup_close_ fd = ret;

	// get property
	ret = get_prop(fd, client_id, name, value, size);
	if (ret != VINF_SUCCESS) {
		disconnect(fd, client_id);
		return ret;
	}

	// disconnect
	ret = disconnect(fd, client_id);
	if (ret != VINF_SUCCESS) {
		// we could ignore the failure, but better to make sure bugs
		// are noticed
		free(*value);
		*value = NULL;
		return ret;
	}

	// for clarity, ensure the Go error return is nil
	errno = 0;
	return 0;
}

int virtualbox_delete_guest_property(char *name) {
	// connect
	uint32_t client_id;
	int ret = start_connection(&client_id);
	if (ret < 0) {
		return ret;
	}
	int _cleanup_close_ fd = ret;

	// delete property
	ret = del_prop(fd, client_id, name);
	if (ret != VINF_SUCCESS) {
		disconnect(fd, client_id);
		return ret;
	}

	// disconnect
	ret = disconnect(fd, client_id);
	if (ret != VINF_SUCCESS) {
		// we could ignore the failure, but better to make sure bugs
		// are noticed
		return ret;
	}

	// for clarity, ensure the Go error return is nil
	errno = 0;
	return 0;
}
