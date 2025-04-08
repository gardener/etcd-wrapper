# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

TOOLS_DIR                  := hack/tools
TOOLS_BIN_DIR              := $(TOOLS_DIR)/bin
GOLANGCI_LINT              := $(TOOLS_BIN_DIR)/golangci-lint
GOSEC                      := $(TOOLS_BIN_DIR)/gosec
GO_ADD_LICENSE             := $(TOOLS_BIN_DIR)/addlicense

# default tool versions
GOLANGCI_LINT_VERSION ?= v1.64.8
GOSEC_VERSION ?= v2.22.2
GO_ADD_LICENSE_VERSION ?= latest

export TOOLS_BIN_DIR := $(TOOLS_BIN_DIR)
export PATH := $(abspath $(TOOLS_BIN_DIR)):$(PATH)

.PHONY: clean-tools-bin
clean-tools-bin:
	rm -rf $(TOOLS_BIN_DIR)/*

#########################################
# Tools                                 #
#########################################

$(GOLANGCI_LINT): $(call tool_version_file,$(GOLANGCI_LINT),$(GOLANGCI_LINT_VERSION))
	@# CGO_ENABLED has to be set to 1 in order for golangci-lint to be able to load plugins
	@# see https://github.com/golangci/golangci-lint/issues/1276
	GOBIN=$(abspath $(TOOLS_BIN_DIR)) CGO_ENABLED=1 go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOSEC): $(call tool_version_file,$(GOSEC),$(GOSEC_VERSION))
	@GOSEC_VERSION=$(GOSEC_VERSION) $(TOOLS_DIR)/install-gosec.sh

$(GO_ADD_LICENSE):
	GOBIN=$(abspath $(TOOLS_BIN_DIR)) go install github.com/google/addlicense@$(GO_ADD_LICENSE_VERSION)