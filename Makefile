REPO_PATH = github.com/coreos/ignition

FMT_PACKAGES = \
    src/config \
    src/exec \
    src/exec/stages \
    src/exec/stages/prepivot \
    src/exec/util \
    src/providers \
    src/providers/cmdline \
    src/providers/util \
    src/registry \

FMT_FILES = \
    src/main.go \

PACKAGES = \
    ./src \
    $(FMT_PACKAGES) \

GFLAGS = \

ABS_PACKAGES = $(PACKAGES:%=$(REPO_PATH)/%)

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
	$(Q)GOPATH=$$(pwd)/gopath go build $(GFLAGS) -o $@ $(REPO)/src

gopath/src/github.com/coreos/ignition:
	$(Q)mkdir --parents $$(dirname $@)
	$(Q)ln --symbolic ../../../.. gopath/src/github.com/coreos/ignition

.PHONY: verify fmt vet fix test
verify: fmt vet fix test
fmt:
	@echo " FMT   $(FMT_PACKAGES) $(FMT_FILES)"
	$(Q)gofmt -l -e -s $(FMT_PACKAGES) $(FMT_FILES)
	$(Q)test -z "$$(gofmt -e -l -s $(FMT_PACKAGES) $(FMT_FILES))"
vet: | gopath/src/github.com/coreos/ignition
	@echo " VET   $(PACKAGES)"
	$(Q)GOPATH=$$(pwd)/gopath go vet $(ABS_PACKAGES)
fix:
	@echo " FIX   $(PACKAGES)"
	$(Q)go tool fix -diff $(PACKAGES)
test: | gopath/src/github.com/coreos/ignition
	@echo " TEST  $(PACKAGES)"
	$(Q)GOPATH=$$(pwd)/gopath go test -cover $(ABS_PACKAGES)
