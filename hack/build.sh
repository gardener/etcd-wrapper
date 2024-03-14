#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

SOURCE_PATH="$(readlink -f "$(dirname "${BASH_SOURCE[0]}")/..")"
BINARY_PATH="${SOURCE_PATH}/bin"
mkdir -p "$BINARY_PATH"

echo "> Build..."

cd "$SOURCE_PATH" &&
  CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) GO111MODULE=on go build \
    -mod vendor \
    -v \
    -o "${BINARY_PATH}"/etcd-wrapper \
    main.go
