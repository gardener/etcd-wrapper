#########################################
# Tools                                 #
#########################################

BIN_DIR := bin
VERSION             := $(shell cat VERSION)
IMAGE_REGISTRY      := europe-docker.pkg.dev/gardener-project/snapshots
IMAGE_REPOSITORY    := $(IMAGE_REGISTRY)/gardener/etcd-wrapper
IMAGE_TAG           := $(VERSION)
IMAGE               := $(IMAGE_REPOSITORY):$(IMAGE_TAG)
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
	@docker buildx build --platform=$(PLATFORM) -t $(IMAGE) -f Dockerfile --rm .

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-build'"; false; fi
	@docker push $(IMAGE)

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
