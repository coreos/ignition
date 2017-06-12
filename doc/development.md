# Development

A Go 1.7+ environment and the `blkid.h` headers are required.

```sh
# Debian/Ubuntu
sudo apt-get install libblkid-dev

# RPM-based
sudo dnf install libblkid-devel
```

## Vendor

Install [glide](https://github.com/Masterminds/glide) and [glide-vc](https://github.com/sgotti/glide-vc) to manage dependencies in the `vendor` directory.

```sh
go get -u github.com/Masterminds/glide
go get -u github.com/sgotti/glide-vc
```

Edit the `glide.yaml` file to update a dependency or add a new dependency. Then make vendor.

```sh
make vendor
```
