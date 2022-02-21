ARG GOLANG_VERSION=1.17.2
ARG BASE_DIST=caden/nvidia_k8s-device-plugin:v0.9.0
ARG PLUGIN_VERSION=v0.2.0

FROM golang:$GOLANG_VERSION as builder

WORKDIR /go/src/github.com/caden2016/nvidia-gpu-scheduler
COPY . .

RUN GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on go build -mod=vendor -ldflags="-s -w -X 'github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app.version=$PLUGIN_VERSION'" ./cmd/gpuserver-ds

FROM $BASE_DIST
COPY ./LICENSE ./licenses/LICENSE
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/github.com/caden2016/nvidia-gpu-scheduler/gpuserver-ds /usr/bin/gpuserver-ds
ENTRYPOINT ["gpuserver-ds"]