#########################################
# Tools                                 #
#########################################

#TOOLS_DIR := hack/tools

REPO_ROOT           := $(shell dirname "$(realpath $(lastword $(MAKEFILE_LIST)))")
BIN_DIR := bin

include $(REPO_ROOT)/hack/tools.mk

.PHONY: add-license-headers
add-license-headers: $(GO_ADD_LICENSE)
	@"$(REPO_ROOT)/hack/add_license_headers.sh"

.PHONY: build
build:
	@"$(REPO_ROOT)/hack/build.sh"

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)/

#-count=1 needed so that test results are not cached
.PHONY: test
test: $(CFSSL)
	@"$(REPO_ROOT)/hack/test.sh" ./cmd/... ./internal/...

.PHONY: revendor
revendor:
	@env GO111MODULE=on go mod tidy
	@env GO111MODULE=on go mod vendor

.PHONY: check
check: $(GOLANGCI_LINT)
	@"$(REPO_ROOT)/hack/check.sh" --golangci-lint-config=./.golangci.yaml ./internal/...