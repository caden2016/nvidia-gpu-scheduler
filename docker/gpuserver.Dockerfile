ARG GOLANG_VERSION=1.17.2
ARG BASE_DIST=ubuntu:21.10
ARG PLUGIN_VERSION=v0.1.0

FROM golang:${GOLANG_VERSION} as builder

WORKDIR /build
COPY . .

RUN GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on go build -mod=vendor -ldflags="-s -w -X 'main.Version=${PLUGIN_VERSION}'" ./cmd/gpuserver

FROM ${BASE_DIST}
COPY ./LICENSE ./licenses/LICENSE
#COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=builder /build/gpuserver /usr/bin/gpuserver
ENTRYPOINT ["gpuserver"]