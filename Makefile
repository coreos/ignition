export GO111MODULE=on

.PHONY: all
all:
	./build

.PHONY: vendor
vendor:
	@go mod vendor
