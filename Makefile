#########################################
# Tools                                 #
#########################################

#TOOLS_DIR := hack/tools
include hack/tools.mk

BIN_DIR := bin

.PHONY: add-license-headers
add-license-headers: $(GO_ADD_LICENSE)
	@./hack/addlicenseheaders.sh ${YEAR}

.PHONY: build-local
build-local:
	.ci/build
	#go build -o bin/etcd-wrapper

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)/

#-count=1 needed so that test results are not cached
.PHONY: test
test:
	go test ./internal/... -count=1

.PHONY: test-cov
test-cov:
	go test ./internal/... -cover

.PHONY: revendor
revendor:
	go mod tidy
	go mod vendor

.PHONY: check
check:
	.ci/check