ARG GOLANG_VERSION=1.17.2
ARG BASE_DIST=ubuntu:21.10
ARG PLUGIN_VERSION=v0.2.0

FROM golang:$GOLANG_VERSION as builder

WORKDIR /go/src/github.com/caden2016/nvidia-gpu-scheduler
COPY . .

RUN GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on go build -mod=vendor -ldflags="-s -w -X 'github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app.version=$PLUGIN_VERSION'" ./cmd/gpuserver

FROM $BASE_DIST
COPY ./LICENSE ./licenses/LICENSE
#COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=builder /go/src/github.com/caden2016/nvidia-gpu-scheduler/gpuserver /usr/bin/gpuserver
ENTRYPOINT ["gpuserver"]