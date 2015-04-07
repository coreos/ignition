REPO_PATH = github.com/coreos/ignition

FMT_PACKAGES = \
    config \
    exec \
    exec/stages \
    exec/stages/prepivot \
    exec/util \
    providers \
    providers/cmdline \
    providers/util \
    registry \

FMT_FILES = \
    main.go \

PACKAGES = \
    . \
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

.PHONY: bin/ignition
bin/ignition: | gopath/src/github.com/coreos/ignition
	@echo " GO    $@"
	$(Q)GOPATH=$$(pwd)/gopath go build $(GFLAGS) -o $@

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
	$(Q)GOPATH=$$(pwd)/gopath go test $(ABS_PACKAGES)
