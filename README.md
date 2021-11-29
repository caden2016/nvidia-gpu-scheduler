# NVIDIA device scheduler extender for Kubernetes
English | [简体中文](./README-zh_CN.md)
## Table of Contents

- [Introduction](#Introduction)
- [Features and Components](#Features-and-Components)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Building and Running Locally](#building-and-running-locally)
- [Versioning](#versioning)


## Introduction

With the help of [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin#readme) and kubernetes kubelet deviceplugin manager, we can schedule our pod by gpu numbers.
But in some case, our node have more gpu devices with different model, we wish kubernetes to shcedule the pod (need 2 gpu with model x) to the nodes which satisfied it. [nvidia-gpu-scheduler](https://github.com/caden2016/nvidia-gpu-scheduler/blob/master/README.md) helps to achieve it and also helps to monitor pods used differnet gpus and gpuinfos of each node.

## Features and Components
### Features
- Real-time data acquisition.(Data will be published in time no matter the gpuserver is restart or the gpuserver-ds of each node is restarted.)
- Health check in time. (the gpuserver notice the health of each node in time with the probe from the gpuserver-ds.)
- Schedule ExtendPoint Filter,Score,Preempt.(Filter nodes with annotation `nvidia.com/gpu.model` of requested pod, scores by gpu numbers of the request model in each node.)
### Components
The NVIDIA device scheduler extender for Kubernetes contains a StatefulSet (gpuserver) and a Daemonset (gpuserver-ds):
#### gpuserver
Provide following apis to help monitor gpu pod and gpu node info:
* GET /apis/metrics.nvidia.com/v1alpha1/podresources
* GET /apis/metrics.nvidia.com/v1alpha1/podresources?watch=true
* GET /apis/metrics.nvidia.com/v1alpha1/gpuinfos

- Help monitor which container of pod is using gpus in kubernetes.
- Help monitor gpu info of each node in kubernetes.
- Help schedule pod with different gpu model needed by extending kubernetes api through the APIService as a kubernetes HTTPExtender server.


#### gpuserver-ds
Populate node gpu devices info to gpuserver.
- It gets pods used gpu device infos with the help of kubelet grpc Server PodResourcesServer
- It gets gpu device infos with the help of [NVML](https://github.com/NVIDIA/go-nvml/blob/master/README.md).

> **_Please note that:_** **You needn't have to do the following extensions when making sure each of your cluster node have only one type of gpu model. If you have more than one type of gpu device in your kubelet node. In order to make the pod scheduled to the kubelet get gpu with model it needs, the following tow need to be changed additionally.**
- The original kubernetes kubelet component is not support to shcedule pod with different gpu model, we need to change it.
- The original [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin#readme) need to be changed, to add gpu model info to kubelet via changing the [kubelet deviceplugin API](https://github.com/kubernetes/kubelet/blob/master/pkg/apis/deviceplugin/v1beta1/api.proto).

## Prerequisites

The list of prerequisites for running the NVIDIA device scheduler extender described below:
* [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin#readme).
* Kubernetes >= v1.13 (gpuserver-ds get pod gpu info base on [kubelet podresources API](https://github.com/kubernetes/kubelet/blob/master/pkg/apis/podresources/v1alpha1/api.proto).)

## Quick Start
1. ### Build with docker.
```shell
$ make all REGISTRY=docker.io/<yourname>
```
2. ### Add an extender configuration to kubernetes kube-scheduler config file.
 ```shell
$ cat kube-scheduler-config.yaml
apiVersion: kubescheduler.config.k8s.io/v1alpha2
...
extenders:
  - urlPrefix: 'https://<kube-apiserver>:6443/apis/metrics.nvidia.com/v1alpha1/schedule'
    filterVerb: filter
    prioritizeVerb: prioritize
    preemptVerb: preempt
    weight: 1
    enableHttps: true
    nodeCacheCapable: true
    ignorable: true
    TLSConfig:
      CAFile: /etc/kubernetes/ssl/ca.pem
      CertFile: /etc/kubernetes/ssl/admin.pem
      KeyFile: /etc/kubernetes/ssl/admin-key.pem
profiles:
- schedulerName: default-scheduler
```
3. ### Deploy with `helm`
Current version of `nvidia-gpu-scheduler` is `v0.1.0`.
The preferred way to deploy it is using `helm`.

Instructions for installing `helm` can be found [here](https://helm.sh/docs/intro/install/).
The simple guide for `helm with nvidia-gpu-scheduler repo` can be found [here](https://caden2016.github.io/nvidia-gpu-scheduler)
* Add and Update chart repo
```shell
# helm repo add ngs https://caden2016.github.io/nvidia-gpu-scheduler
# helm repo update
```
* Install from chart repo，xxx is the release name. nodeinfo=gpu is the label of gpu node, where to deploy gpuserver-ds.
```shell
# helm install xxx ngs/nvidia-gpu-scheduler --version 0.1.0 --namespace kube-system  --set nodeSelectorDaemonSet.nodeinfo=gpu
# helm  list --namespace kube-system
```

## Building and Running Locally

## Versioning
Use the versioning to follow [SEMVER](https://semver.org/). The first version following this scheme has been tagged `v0.0.0`.

Going forward, the major version of the `nvidia-gpu-scheduler` will only change
following a change in the [kubelet podresources API](https://github.com/kubernetes/kubelet/blob/master/pkg/apis/podresources/v1alpha1/api.proto) itself.
For example, version `v1alpha1` of `kubelet podresources API` corresponds to version `v0.x.x` of `nvidia-gpu-scheduler`.
If a new `v2beta2` version of `kubelet podresources API` comes out, then `nvidia-gpu-scheduler` will increase its major version to `1.x.x`.

As of now, the podresources API for Kubernetes >= v1.13 is `v1alpha1` or `v1` added compatibly.  If you
have a version of Kubernetes >= 1.13 you can deploy any `nvidia-gpu-scheduler` version >
`v0.0.0`.