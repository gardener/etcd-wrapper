#!/usr/bin/env bash

set -e

SOURCE_PATH="$(readlink -f "$(dirname "${BASH_SOURCE[0]}")/..")"
BINARY_PATH="${SOURCE_PATH}/bin"
if [[ -z "${BINARY_PATH}" ]]; then
  mkdir -p "$BINARY_PATH"
fi

echo "> Build..."

cd "$SOURCE_PATH" &&
  CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) GO111MODULE=on go build \
    -mod vendor \
    -v \
    -o "${BINARY_PATH}"/etcd-wrapper \
    main.go
