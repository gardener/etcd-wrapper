FROM golang:1.20 as builder
WORKDIR /go/src/github.com/gardener/etcd-wrapper
COPY . .

# # cache deps before building and copying source so that we don't need to re-download as much
# # and so that source changes don't invalidate our downloaded layer
#RUN go mod download

# Build
RUN make build

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian11:latest AS wrapper
WORKDIR /
COPY --from=builder /go/src/github.com/gardener/etcd-wrapper/bin/etcd-wrapper /etcd-wrapper
ENTRYPOINT ["/etcd-wrapper"]
