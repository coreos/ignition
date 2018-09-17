.PHONY: all
all:
	./build

.PHONY: vendor
vendor:
	@glide --quiet update --strip-vendor
	@glide-vc --use-lock-file --no-tests --only-code
