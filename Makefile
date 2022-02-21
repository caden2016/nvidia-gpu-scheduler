.PHONY:  vet fmt build clean gpuserver gpuserver-ds all save push

DOCKER   ?= docker
GOLANG_VERSION ?= 1.17.2
BASE_DIST ?= ubuntu:21.10
BASE_DIST_DS ?= caden/nvidia_k8s-device-plugin:v0.9.0

MAIN_VERSION  ?= v0.2.0
COMMIT_DATE=$(shell git --no-pager log -1 --format='%cI')
COMMIT_ID=$(shell git rev-parse --short HEAD)
PLUGIN_VERSION=$(MAIN_VERSION)-$(COMMIT_ID)-$(COMMIT_DATE)

REGISTRY ?= docker.io/caden
IMAGE_NAME := $(REGISTRY)/gpuserver
OUT_IMAGE = $(IMAGE_NAME):$(MAIN_VERSION)
IMAGE_NAME_DS := $(REGISTRY)/gpuserver-ds
OUT_IMAGE_DS = $(IMAGE_NAME_DS):$(MAIN_VERSION)

IMAGE_TAR ?= nvidia-gpu-scheduler.img
OUTPUT_DIR ?= _output/bin/

all: gpuserver gpuserver-ds
	echo "success build images: "$(OUT_IMAGE)" and "$(OUT_IMAGE_DS)

fmt:
	go list -f '{{.Dir}}' ./... \
		| xargs gofmt -s -l -w

vet:
	go vet ./...

clean:
	rm -rf $(OUTPUT_DIR)*

build:
	echo "commitID:${COMMIT_ID},commitDate:${COMMIT_DATE}"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
	go build -mod=vendor -ldflags="-s -w -X 'github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app.version=${PLUGIN_VERSION}'" -o $(OUTPUT_DIR) ./cmd/gpuserver
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on \
	go build -mod=vendor -ldflags="-s -w -X 'github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app.version=${PLUGIN_VERSION}'" -o $(OUTPUT_DIR) ./cmd/gpuserver-ds

gpuserver:
	$(DOCKER) rmi $(OUT_IMAGE) || true
	$(DOCKER) build \
		--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
		--build-arg BASE_DIST=$(BASE_DIST) \
		--build-arg PLUGIN_VERSION=$(PLUGIN_VERSION) \
		--tag $(OUT_IMAGE) \
		--file docker/gpuserver.Dockerfile \
		.

gpuserver-ds:
	$(DOCKER) rmi $(OUT_IMAGE_DS) || true
	$(DOCKER) build \
		--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
		--build-arg BASE_DIST=$(BASE_DIST_DS) \
		--build-arg PLUGIN_VERSION=$(PLUGIN_VERSION) \
		--tag $(OUT_IMAGE_DS) \
		--file docker/gpuserver-ds.Dockerfile \
		.

save:
	$(DOCKER) inspect $(OUT_IMAGE) > /dev/null
	$(DOCKER) inspect $(OUT_IMAGE_DS) > /dev/null
	$(DOCKER) save -o $(IMAGE_TAR) $(OUT_IMAGE) $(OUT_IMAGE_DS)

push:
	$(DOCKER) inspect $(OUT_IMAGE) > /dev/null
	$(DOCKER) inspect $(OUT_IMAGE_DS) > /dev/null
	$(DOCKER) push $(OUT_IMAGE)
	$(DOCKER) push $(OUT_IMAGE_DS)


