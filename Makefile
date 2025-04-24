#########################################
# Tools                                 #
#########################################

BIN_DIR := bin
VERSION             := $(shell cat VERSION)
IMAGE_PATH          := europe-docker.pkg.dev/gardener-project/snapshots/gardener/etcd-wrapper
IMG                 := $(IMAGE_PATH):$(VERSION)
PLATFORM            ?= $(shell docker info --format '{{.OSType}}/{{.Architecture}}')

include hack/tools.mk

.PHONY: add-license-headers
add-license-headers: $(GO_ADD_LICENSE)
	@./hack/add_license_headers.sh

.PHONY: build
build:
	@./hack/build.sh

.PHONY: docker-build
docker-build:
	@docker buildx build --platform=$(PLATFORM) -t $(IMG) -f Dockerfile --rm .

.PHONY: docker-push
docker-push:
	@docker push $(IMG)

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)/

.PHONY: test
test:
	@./hack/test.sh ./cmd/... ./internal/...

.PHONY: revendor
revendor:
	@env GO111MODULE=on go mod tidy
	@env GO111MODULE=on go mod vendor

.PHONY: check
check: $(GOLANGCI_LINT)
	@./hack/check.sh --golangci-lint-config=./.golangci.yaml ./internal/...

.PHONY: sast
sast: $(GOSEC)
	@./hack/sast.sh

.PHONY: sast-report
sast-report: $(GOSEC)
	@./hack/sast.sh --gosec-report true
