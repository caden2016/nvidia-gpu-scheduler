# NVIDIA device scheduler extender for Kubernetes
[English](./README.md) | 简体中文
## Table of Contents

- [介绍](#介绍)
- [特征和组件](#特征和组件)
- [先决条件](#先决条件)
- [快速开始](#快速开始)
- [本地构建和运行](#本地构建和运行)
- [版本控制](#版本控制)


## 介绍

在英伟达官方k8s插件 [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin#readme) 的帮助下，我们可以通过pod请求固定数目的gpu并自动调度到拥有足够请求gpu数目的节点上。但是在某种场景下，比如我们不同的节点有多个类型的gpu。这种情况下我们希望pod（需要2个x类型的gpu）能够自动调度到满足请求类型和个数的节点上。[nvidia-gpu-scheduler](https://github.com/caden2016/nvidia-gpu-scheduler/blob/master/README.md) 可以实现该场景的需求并提供了基于pod内部容器粒度的gpu使用监控和节点gpu信息查询接口。

## 特征和组件
### 特征
- 实时数据采集。（gpuserver都会采集最新的数据，不管gpuserver故障重启或各个节点上的gpuserver-ds故障重启。）
- 实时健康检测。（gpuserver通过gpuserver-ds的探针通知，及时得知每个节点的健康状况。）
- 调度扩展点：Filter,Score,Preempt。（对请求pod注解包含 `nvidia.com/gpu.model`，过滤不符合gpu类型的节点。对每种gpu类型的节点按照gpu个数打分进行优选。）
### 组件
The NVIDIA device scheduler extender for Kubernetes contains a StatefulSet (gpuserver) and a Daemonset (gpuserver-ds):
#### gpuserver
提供以下接口来监控请求gpu的pod和gpu节点的信息：
* GET /apis/metrics.nvidia.com/v1alpha1/podresources
* GET /apis/metrics.nvidia.com/v1alpha1/podresources?watch=true
* GET /apis/metrics.nvidia.com/v1alpha1/gpuinfos

- 帮助监控kubernetes集群内pod中使用gpu的容器情况。
- 帮助监控kubernetes集群内各个节点gpu使用情况。
- 通过APIService扩展kubernetes api。作为kubernetes HTTPExtender服务器，帮助调度不同gpu型号需求的pod。

#### gpuserver-ds
为gpuserver采集节点gpu信息。
- 通过kubelet grpc Server PodResourcesServer采集kubernetes集群中使用gpu的pod信息。
- 通过[NVML](https://github.com/NVIDIA/go-nvml/blob/master/README.md) 采集节点gpu信息。

> **_注意：_** **如果确保你的集群每个节点只有一种gpu类型网卡，不必执行以下扩展。如果你的集群节点拥有多种类型的gpu设备。为了让调度到节点的pod分配到所需类型的gpu设备，必须进行以下扩展。**
- 原始的kubernetes kubelet组件并不支持调度不同gpu类型的pod，需要进行扩展。
- 原始的[NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin#readme) 并没采集gpu类型信息，需要修改[kubelet deviceplugin API](https://github.com/kubernetes/kubelet/blob/master/pkg/apis/deviceplugin/v1beta1/api.proto) 扩展传递给kubelet的gpu信息。

## 先决条件

运行NVIDIA device scheduler extender的先决条件列表如下：
* [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin#readme).
* Kubernetes >= v1.13 (gpuserver-ds 依赖 [kubelet podresources API](https://github.com/kubernetes/kubelet/blob/master/pkg/apis/podresources/v1alpha1/api.proto) 获取pod的gpu使用信息。)

## 快速开始
1. ### 使用docker构建项目。
```shell
$ make all REGISTRY=docker.io/<yourname>
```
2. ### 向kubernetes kube-scheduler添加extenders部分配置。
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
3. ### 使用`helm`部署。
当前`nvidia-gpu-scheduler`版本为`v0.1.0`。最好的安装方式是使用`helm`。

`helm`的安装可以参考 [这里](https://helm.sh/docs/intro/install/) 。
使用`helm`安装`nvidia-gpu-scheduler`的简单指导可以参考[这里](https://caden2016.github.io/nvidia-gpu-scheduler) 。

* 添加和更新helm仓库。
```shell
# helm repo add ngs https://caden2016.github.io/nvidia-gpu-scheduler
# helm repo update
```
* 通过helm仓库安装指定版本应用。xxx为应用实例名称。nodeinfo=gpu为gpuserver-ds部署到gpu节点上的label。
```shell
# helm install xxx ngs/nvidia-gpu-scheduler --version 0.1.0 --namespace kube-system  --set nodeSelectorDaemonSet.nodeinfo=gpu
# helm  list --namespace kube-system
```
## 本地构建和运行

## 版本控制
按照 [SEMVER](https://semver.org/) 进行版本控制。遵循此方案第一个版本被标记为`v0.0.0`。

接下来`nvidia-gpu-scheduler`的主要版本将跟随 [kubelet podresources API](https://github.com/kubernetes/kubelet/blob/master/pkg/apis/podresources/v1alpha1/api.proto) 而改变。
例如 `kubelet podresources API` `v1alpha1`版本对应`nvidia-gpu-scheduler` `v0.x.x`版本。
如果`kubelet podresources API` `v2beta2` 出来了，那么对应`nvidia-gpu-scheduler`的`1.x.x`版本。

到目前为止，`kubelet podresources API` Kubernetes >= v1.13 is `v1alpha1` 或者兼容 `v1`。
如果你的 Kubernetes >= 1.13，你可以部署任何版本大于`v0.0.0`的任何`nvidia-gpu-scheduler`版本。