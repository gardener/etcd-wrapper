# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

#This ops pod is to be used as an ephemeral container while debugging etcd-wrapper

FROM gcr.io/etcd-development/etcd:v3.4.26 as source-amd64
FROM gcr.io/etcd-development/etcd:v3.4.26-arm64 as source-arm64

FROM source-$TARGETARCH as source

FROM alpine:3.15.6

WORKDIR /

RUN apk update \
    && apk --no-cache add \
    bash \
    curl \
    vim \
    && mkdir -p work \
    && chown -R 65532:65532 work

COPY --chown=65532:65532 --chmod=744 ops/print-etcd-cert-paths.sh /usr/local/bin/print-etcd-cert-paths
COPY --from=source /usr/local/bin/etcdctl /usr/local/bin

ENTRYPOINT ["/bin/bash"]