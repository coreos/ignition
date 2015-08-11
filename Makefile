REPO_PATH = github.com/coreos/ignition

PACKAGES = \
    src \
    src/config \
    src/exec \
    src/exec/stages \
    src/exec/stages/disks \
    src/exec/stages/files \
    src/exec/util \
    src/providers \
    src/providers/cmdline \
    src/providers/ec2 \
    src/providers/file \
    src/providers/util \
    src/registry \

GFLAGS = \

GLDFLAGS = \
    -X main.version $(VERSION) \

ABS_PACKAGES = $(PACKAGES:%=$(REPO_PATH)/%)

VERSION = $(shell git describe --dirty)
ifeq ($(VERSION), )
    $(error "failed to determine version")
endif
# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
	BUILD_VERBOSE = $(V)
endif
ifndef BUILD_VERBOSE
	BUILD_VERBOSE = 0
endif

ifeq ($(BUILD_VERBOSE),1)
	Q =
else
	Q = @
endif

.PHONY: all
all: bin/ignition

.PHONY: FORCE
bin/ignition: FORCE | gopath/src/github.com/coreos/ignition

bin/ignition: REPO=github.com/coreos/ignition

bin/%:
	@echo " GO    $@"
	$(Q)GOPATH=$$(pwd)/gopath go build $(GFLAGS) -ldflags "$(GLDFLAGS)" -o $@ $(REPO)/src

gopath/src/github.com/coreos/ignition:
	$(Q)mkdir --parents $$(dirname $@)
	$(Q)ln --symbolic ../../../.. gopath/src/github.com/coreos/ignition

.PHONY: check fmt vet fix test
check: fmt vet fix test
fmt:
	@echo " FMT   src"
	$(Q)FMTOUT=$$(gofmt -d -e -s src); test -z "$$FMTOUT" || { echo "$$FMTOUT" && exit 1; }
vet: | gopath/src/github.com/coreos/ignition
	@echo " VET   $(PACKAGES)"
	$(Q)GOPATH=$$(pwd)/gopath go vet $(ABS_PACKAGES)
fix:
	@echo " FIX   $(PACKAGES)"
	$(Q)go tool fix -diff $(PACKAGES)
test: | gopath/src/github.com/coreos/ignition
	@echo " TEST  $(PACKAGES)"
	$(Q)GOPATH=$$(pwd)/gopath go test -cover $(ABS_PACKAGES)
