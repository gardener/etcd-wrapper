FROM golang:1.23.6 as builder
WORKDIR /go/src/github.com/gardener/etcd-wrapper
COPY . .

# Build
RUN make build
#RUN mkdir -p /var/etcd/data

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian11:nonroot AS wrapper
WORKDIR /
COPY --from=builder /go/src/github.com/gardener/etcd-wrapper/bin/etcd-wrapper /etcd-wrapper
ENTRYPOINT ["/etcd-wrapper"]
