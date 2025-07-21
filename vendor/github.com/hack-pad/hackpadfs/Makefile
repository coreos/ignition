BROWSERTEST_VERSION = v0.8
LINT_VERSION = 1.59.1
GO_BIN = $(shell printf '%s/bin' "$$(go env GOPATH)")
SHELL = bash

current_platform := $(shell go env GOOS)/$(shell go env GOARCH)
# Only use the race detector on supported architectures.
# Go's supported platforms pulled from 'go help build' under '-race'.
race_platforms := linux/amd64 freebsd/amd64 darwin/amd64 darwin/arm64 windows/amd64 linux/ppc64le linux/arm64
RACE_ENABLED = $(if $(findstring ${current_platform},${race_platforms}),true,false)

.PHONY: all
all: lint test

.PHONY: lint-deps
lint-deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GO_BIN}" v${LINT_VERSION}; \
	fi
	@if ! which jsguard >/dev/null; then \
		go install github.com/hack-pad/safejs/jsguard/cmd/jsguard@latest; \
	fi

.PHONY: lint
lint: lint-deps
	"${GO_BIN}/golangci-lint" run
	GOOS=js GOARCH=wasm "${GO_BIN}/golangci-lint" run
	cd examples && "${GO_BIN}/golangci-lint" run --config=../.golangci.yml --timeout=5m
	GOOS=js GOARCH=wasm "${GO_BIN}/jsguard" ./...

.PHONY: test-deps
test-deps:
	@if [ ! -f "${GO_BIN}/go_js_wasm_exec" ]; then \
		set -ex; \
		GOOS= GOARCH= go install github.com/agnivade/wasmbrowsertest@${BROWSERTEST_VERSION}; \
		ln -s "${GO_BIN}/wasmbrowsertest" "${GO_BIN}/go_js_wasm_exec"; \
	fi
	@go install github.com/mattn/goveralls@v0.0.9

.PHONY: test
test: test-deps
	go test .  # Run library-level checks first, for more helpful build tag failure messages.
	go test -race=${RACE_ENABLED} -coverprofile=native-cover.out ./...
	if [[ "$$CI" != true || $$(uname -s) == Linux ]]; then \
		set -ex; \
		GOOS=js GOARCH=wasm go test -coverprofile=js-cover.out -covermode=atomic ./...; \
		cd examples && go test -race=${RACE_ENABLED} ./...; \
	fi
	{ echo 'mode: atomic'; cat *-cover.out | grep -v '^mode:'; } > cover.out && rm *-cover.out
	go tool cover -func cover.out | grep total:
	@if [[ "$$CI" == true && $$(uname -sm) == 'Linux x86_64' && "$$(go version)" == *go"$$COVERAGE_VERSION"* ]]; then \
		set -ex; \
		goveralls -coverprofile=cover.out -service=github || true; \
	fi
