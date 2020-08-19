export GO111MODULE=on

# Canonical version of this in https://github.com/coreos/coreos-assembler/blob/6eb97016f4dab7d13aa00ae10846f26c1cd1cb02/Makefile#L19
GOARCH:=$(shell uname -m)
ifeq ($(GOARCH),x86_64)
	GOARCH=amd64
else ifeq ($(GOARCH),aarch64)
	GOARCH=arm64
else ifeq ($(patsubst armv%,arm,$(GOARCH)),arm)
	GOARCH=arm
else ifeq ($(patsubst i%86,386,$(GOARCH)),386)
	GOARCH=386
endif

.PHONY: all
all:
	./build

.PHONY: install
install: all
	for x in dracut/*; do \
	  bn=$$(basename $$x); \
	  install -m 0644 -D -t $(DESTDIR)/usr/lib/dracut/modules.d/$${bn} $$x/*; \
	done
	chmod a+x $(DESTDIR)/usr/lib/dracut/modules.d/*/*.sh $(DESTDIR)/usr/lib/dracut/modules.d/*/*-generator
	install -m 0644 -D -t $(DESTDIR)/usr/lib/systemd/system systemd/*
	install -m 0755 -D -t $(DESTDIR)/usr/lib/dracut/modules.d/30ignition bin/$(GOARCH)/ignition
	install -m 0755 -D -t $(DESTDIR)/usr/bin bin/$(GOARCH)/ignition-validate

.PHONY: vendor
vendor:
	@go mod vendor
	@go mod tidy
