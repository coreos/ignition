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

FMT_FILES = \
    main.go \

PACKAGES = \
    . \
    $(FMT_PACKAGES) \

GFLAGS = \

ABS_PACKAGES = $(PACKAGES:%=$(REPO_PATH)/%)

.PHONY: all
all: bin/ignition

.PHONY: bin/ignition
bin/ignition: | gopath/src/github.com/coreos/ignition
	@echo " GO    $@"
	@GOPATH=$$(pwd)/gopath go build $(GFLAGS) -o $@

gopath/src/github.com/coreos/ignition:
	@mkdir --parents $$(dirname $@)
	@ln --symbolic ../../../.. gopath/src/github.com/coreos/ignition

.PHONY: verify fmt vet fix test
verify: fmt vet fix test
fmt:
	@echo " FMT   $(FMT_PACKAGES) $(FMT_FILES)"
	@gofmt -l -e -s $(FMT_PACKAGES) $(FMT_FILES)
	@test -z "$$(gofmt -e -l -s $(FMT_PACKAGES) $(FMT_FILES))"
vet: | gopath/src/github.com/coreos/ignition
	@echo " VET   $(PACKAGES)"
	@GOPATH=$$(pwd)/gopath go vet $(ABS_PACKAGES)
fix:
	@echo " FIX   $(PACKAGES)"
	@go tool fix -diff $(PACKAGES)
test: | gopath/src/github.com/coreos/ignition
	@echo " TEST  $(PACKAGES)"
	@GOPATH=$$(pwd)/gopath go test $(ABS_PACKAGES)
