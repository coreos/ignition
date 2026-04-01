export GO111MODULE=on

INSTALL_ALL_DRACUT_MODULES=0

# Canonical version of this in https://github.com/coreos/coreos-assembler/blob/6eb97016f4dab7d13aa00ae10846f26c1cd1cb02/Makefile#L19
GOARCH:=$(shell uname -m)
ifeq ($(GOARCH),x86_64)
	GOARCH=amd64
else ifeq ($(GOARCH),aarch64)
	GOARCH=arm64
else ifeq ($(GOARCH),loongarch64)
	GOARCH=loong64
else ifeq ($(patsubst armv%,arm,$(GOARCH)),arm)
	GOARCH=arm
else ifeq ($(patsubst i%86,386,$(GOARCH)),386)
	GOARCH=386
endif
export GOARCH

BIN_PATH ?= bin
export BIN_PATH

.PHONY: all
all: ignition ignition-validate ignition-validate-cross butane butane-cross

.PHONY: ignition
ignition:
	./build ignition

.PHONY: ignition-validate
ignition-validate:
	./build ignition-validate

.PHONY: ignition-validate-cross
ignition-validate-cross:
	./build ignition-validate-cross

.PHONY: butane
butane:
	./build butane

.PHONY: butane-cross
butane-cross:
	./build butane-cross

.PHONY: install
install:
	if [ $(INSTALL_ALL_DRACUT_MODULES) -eq 1 ]; then \
		for x in dracut/*; do \
			bn=$$(basename $$x); \
			install -m 0644 -D -t $(DESTDIR)/usr/lib/dracut/modules.d/$${bn} $$x/*; \
		done; \
		chmod a+x $(DESTDIR)/usr/lib/dracut/modules.d/40ignition-ostree/ignition-relabel; \
	else \
		install -m 0644 -D -t $(DESTDIR)/usr/lib/dracut/modules.d/30ignition dracut/30ignition/*; \
	fi

	chmod a+x $(DESTDIR)/usr/lib/dracut/modules.d/*/*.sh $(DESTDIR)/usr/lib/dracut/modules.d/*/*-generator
	install -m 0644 -D -t $(DESTDIR)/usr/lib/systemd/system systemd/ignition-delete-config.service
	install -m 0755 -D -t $(DESTDIR)/usr/lib/dracut/modules.d/30ignition $(BIN_PATH)/ignition
	install -m 0755 -D -t $(DESTDIR)/usr/bin $(BIN_PATH)/ignition-validate
	install -m 0755 -d $(DESTDIR)/usr/libexec
	ln -sf ../lib/dracut/modules.d/30ignition/ignition $(DESTDIR)/usr/libexec/ignition-apply
	ln -sf ../lib/dracut/modules.d/30ignition/ignition $(DESTDIR)/usr/libexec/ignition-rmcfg

# For distros that need to build cross platform ignition-validate binaries
# Used in fedora for now. See ignition rpm spec for details.
.PHONY: install-ignition-validate-cross
install-ignition-validate-cross:
	install -d -p $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-aarch64-apple-darwin $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-aarch64-unknown-linux-gnu-static $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-ppc64le-unknown-linux-gnu-static $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-s390x-unknown-linux-gnu-static $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-x86_64-apple-darwin $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-x86_64-pc-windows-gnu.exe $(DESTDIR)/usr/share/ignition
	install -p -m 0644 $(BIN_PATH)/ignition-validate-x86_64-unknown-linux-gnu-static $(DESTDIR)/usr/share/ignition

.PHONY: install-butane
install-butane:
	install -m 0755 -D -t $(DESTDIR)/usr/bin $(BIN_PATH)/butane

# For distros that need to build cross platform butane binaries
.PHONY: install-butane-cross
install-butane-cross:
	install -d -p $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-aarch64-apple-darwin $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-aarch64-unknown-linux-gnu-static $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-ppc64le-unknown-linux-gnu-static $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-s390x-unknown-linux-gnu-static $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-x86_64-apple-darwin $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-x86_64-pc-windows-gnu.exe $(DESTDIR)/usr/share/butane
	install -p -m 0644 $(BIN_PATH)/butane-x86_64-unknown-linux-gnu-static $(DESTDIR)/usr/share/butane

.PHONY: install-grub-for-bootupd
install-grub-for-bootupd:
	install -d -p $(DESTDIR)/usr/lib/bootupd/grub2-static/configs.d
	install -m 0644 -D -t $(DESTDIR)/usr/lib/bootupd/grub2-static/configs.d grub2/05_ignition.cfg

.PHONY: vendor
vendor:
	@go mod vendor
	@go mod tidy
