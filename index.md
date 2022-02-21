# Helm Guide for github

## [中文]|[[English](index_en.md)]
## helm的chart仓库地址为：https://caden2016.github.io/nvidia-gpu-scheduler

## Chart仓库的使用方法

0. helm安装和命令补全
```bash
# wget https://get.helm.sh/helm-v3.7.2-linux-amd64.tar.gz 
# source <(helm completion bash)
```

1. 添加和更新chart仓库
```bash
# helm repo add myrepo https://caden2016.github.io/nvidia-gpu-scheduler
# helm repo update
```

2. 查看chart仓库列表
```bash
# helm repo list
NAME  	URL                                   
myrepo	https://caden2016.github.io/nvidia-gpu-scheduler
```

3. 搜索chart包
```bash
# helm search repo myrepo
NAME                              	CHART VERSION	APP VERSION	DESCRIPTION                                            
myrepo/nvidia-gpu-scheduler       	0.2.0        	0.2.0      	A Helm chart for nvidia-gpu-scheduler on Kubernetes 
```

4. 安装chart包，xxx为relaese名字，nodeinfo=gpu为gpuserver-ds部署到gpu节点上的label。
```bash
# helm install xxx myrepo/nvidia-gpu-scheduler --version 0.2.0 --namespace kube-system  --set nodeSelectorDaemonSet.nodeinfo=gpu
# helm  list --all-namespaces
# helm uninstall xxx --namespace kube-system
```

5. 删除chart仓库
```bash
# helm repo remove myrepo
```
