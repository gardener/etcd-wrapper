#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# create the testdata directory to all PKI resources that are created. All unit tests will refer
# to PKI resources kept under this path.
TEST_PKI_DIR="${PROJECT_DIR}/internal/testdata"
echo "TEST_PKI_DIR = $TEST_PKI_DIR"
mkdir -p "${TEST_PKI_DIR}"

echo "> Test..."

echo "> Running tests..."
go test -v -coverprofile cover.out "$@"
go tool cover -func=cover.out
