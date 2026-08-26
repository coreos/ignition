.DEFAULT_GOAL = test

extension = $(patsubst windows,.exe,$(filter windows,$(1)))
GOOS ?= $(shell go version | sed 's/^.*\ \([a-z0-9]*\)\/\([a-z0-9]*\)/\1/')
GOARCH ?= $(shell go version | sed 's/^.*\ \([a-z0-9]*\)\/\([a-z0-9]*\)/\2/')

ifeq ("$(TARGETVARIANT)","")
ifneq ("$(GOARM)","")
TARGETVARIANT := v$(GOARM)
endif
else
ifeq ("$(GOARM)","")
GOARM ?= $(subst v,,$(TARGETVARIANT))
endif
endif

ifeq ("$(CI)","true")
LINT_PROCS ?= 1
# don't write coverage profile in CI - allows Go's test result cache to work
TEST_COVER_ARGS=
else
LINT_PROCS ?= $(shell nproc)
TEST_COVER_ARGS=-coverprofile=c.out
endif

# test with race detector on supported platforms
# windows/amd64 is supported in theory, but in practice it requires a C compiler
race_platforms := 'linux/amd64' 'darwin/amd64' 'darwin/arm64'
ifeq (,$(findstring '$(GOOS)/$(GOARCH)',$(race_platforms)))
export CGO_ENABLED=0
TEST_ARGS=
else
TEST_ARGS=-race
endif

test:
	go test $(TEST_ARGS) $(TEST_COVER_ARGS) ./...

bench.txt:
	go test -benchmem -run=xxx -bench . ./... | tee $@

bin/fscli_%v7$(call extension,$(GOOS)): $(shell find . -type f -name "*.go")
	GOOS=$(shell echo $* | cut -f1 -d-) \
	GOARCH=$(shell echo $* | cut -f2 -d- ) \
	GOARM=7 \
	CGO_ENABLED=0 \
		go build $(BUILD_ARGS) -o $@ ./examples/fscli

bin/fscli_windows-%.exe: $(shell find . -type f -name "*.go")
	GOOS=windows \
	GOARCH=$* \
	GOARM= \
	CGO_ENABLED=0 \
		go build $(BUILD_ARGS) -o $@ ./examples/fscli

bin/fscli_%$(TARGETVARIANT)$(call extension,$(GOOS)): $(shell find . -type f -name "*.go")
	GOOS=$(shell echo $* | cut -f1 -d-) \
	GOARCH=$(shell echo $* | cut -f2 -d- ) \
	GOARM=$(GOARM) \
	CGO_ENABLED=0 \
		go build $(BUILD_ARGS) -o $@ ./examples/fscli

bin/fscli: bin/fscli_$(GOOS)-$(GOARCH)
	cp $< $@

lint:
	@golangci-lint run --verbose --max-same-issues=0 --max-issues-per-linter=0

ci-lint:
	@golangci-lint run --verbose --max-same-issues=0 --max-issues-per-linter=0 --out-format=github-actions

ifneq ("$(VERSION)","")
release:
	git tag -sm "Releasing v$(VERSION)" v$(VERSION) && git push --tags
	gh release create v$(VERSION) --draft --generate-notes
endif

# this is a special target for testing a package on Windows from a non-Windows
# host. It builds the Windows test binary, then SCPs it to the Windows host, and
# runs the tests there. This depends on the GO_REMOTE_WINDOWS environment
# variable being set as 'username@host'. The Windows host must have Git Bash
# installed, or maybe MSYS2, so that a number of standard Unix tools are
# available. Git must also be configured with a username and email address. See
# the GitHub workflow config in .github/workflows/build.yml for hints.
# A recent PowerShell is also required, such as version 7.3 or later.
#
# An F: drive is expected to be available, with a tmp directory. This is used
# to make sure we can deal with files on a different volume.
.SECONDEXPANSION:
$(shell go list -f '{{ if not (eq "" (join .TestGoFiles "")) }}testbin/{{.ImportPath}}.test.exe.remote{{end}}' ./...): $$(shell go list -f '{{.Dir}}' $$(subst testbin/,,$$(subst .test.exe.remote,,$$@)))
	@echo $<
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -tags timetzdata -c -o ./testbin/remote-test.exe $<
	@scp -q ./testbin/remote-test.exe $(GO_REMOTE_WINDOWS):/$(shell ssh $(GO_REMOTE_WINDOWS) 'echo %TEMP%' | cut -f2 -d= | sed -e 's#\\#/#g')/
	@ssh -o 'SetEnv TMP=F:\tmp' $(GO_REMOTE_WINDOWS) '%TEMP%\remote-test.exe'

# test-remote-windows runs the above target for all packages that have tests
test-remote-windows: $(shell go list -f '{{ if not (eq "" (join .TestGoFiles "")) }}testbin/{{.ImportPath}}.test.exe.remote{{end}}' ./...)


.PHONY: test lint ci-lint
.DELETE_ON_ERROR:
.SECONDARY:
