ARG GOLANG_VERSION=1.17.2
ARG BASE_DIST=caden/nvidia_k8s-device-plugin:v0.9.0
ARG PLUGIN_VERSION=v0.1.0

FROM golang:${GOLANG_VERSION} as builder

WORKDIR /build
COPY . .

RUN GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on go build -mod=vendor -ldflags="-s -w -X 'main.Version=${PLUGIN_VERSION}'" ./cmd/gpuserver-ds

FROM ${BASE_DIST}
COPY ./LICENSE ./licenses/LICENSE
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /build/gpuserver-ds /usr/bin/gpuserver-ds
ENTRYPOINT ["gpuserver-ds"]