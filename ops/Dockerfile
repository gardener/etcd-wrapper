# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

#This ops pod is to be used as an ephemeral container while debugging etcd-wrapper

FROM gcr.io/etcd-development/etcd:v3.4.13 as source-amd64
FROM gcr.io/etcd-development/etcd:v3.4.13-arm64 as source-arm64

FROM source-$TARGETARCH as source

FROM alpine:3.15.6

WORKDIR /

RUN apk add --update bash curl

COPY --from=source /usr/local/bin/etcdctl /usr/local/bin

ENTRYPOINT ["/bin/sh"]