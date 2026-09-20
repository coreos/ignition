#!/usr/bin/env sh

mise exec go@1.26.5 -- ./build butane
GOMPLATE="$(mise which gomplate --tool=gomplate@5.2.0)"
POC_NAME=greeting POC_MESSAGE="rendered from a local file" \
  bin/butane \
  --file-processor "$GOMPLATE" \
  --files-dir poc/file-processor \
  poc/file-processor/config.bu \
  --output poc/file-processor/config.ign
