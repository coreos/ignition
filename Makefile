export GO111MODULE=on

# Canonical version of this in https://github.com/coreos/coreos-assembler/blob/6eb97016f4dab7d13aa00ae10846f26c1cd1cb02/Makefile#L19
GOARCH:=$(shell uname -m)
ifeq ($(GOARCH),x86_64)
	GOARCH=amd64
else ifeq ($(GOARCH),aarch64)
	GOARCH=arm64
endif

.PHONY: all
all:
	./build

# This currently assumes you're using https://github.com/coreos/ignition-dracut/
# If in the future any other initramfs integration appears, feel free to add a PR
# to make this configurable.
.PHONY: install
install: all
	install -m 0755 -D -t $(DESTDIR)/usr/lib/dracut/modules.d/30ignition bin/$(GOARCH)/ignition
	install -m 0755 -D -t $(DESTDIR)/usr/bin bin/$(GOARCH)/ignition-validate

.PHONY: vendor
vendor:
	@go mod vendor
	@go mod tidy
