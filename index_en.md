# Helm Guide for github

## [[中文](index.md)]|[English]
## The repo url of helm chart: https://caden2016.github.io/nvidia-gpu-scheduler

## Usage of Helm Repo Chart

0. helm installation and command completion
```bash
# wget https://get.helm.sh/helm-v3.7.2-linux-amd64.tar.gz 
# source <(helm completion bash)
```

1. Add and Update chart repo
```bash
# helm repo add myrepo https://caden2016.github.io/nvidia-gpu-scheduler
# helm repo update
```

2. List chart repo
```bash
# helm repo list
NAME  	URL                                   
myrepo	https://caden2016.github.io/nvidia-gpu-scheduler
```

3. Search chart repo
```bash
# helm search repo myrepo
NAME                              	CHART VERSION	APP VERSION	DESCRIPTION                                            
myrepo/nvidia-gpu-scheduler       	0.1.0        	0.1.0      	A Helm chart for nvidia-gpu-scheduler on Kubernetes 
```

4. Install from chart repo， xxx is the release name. nodeinfo=gpu is the label of gpu node, where to deploy gpuserver-ds.
```bash
# helm install xxx myrepo/nvidia-gpu-scheduler --version 0.1.0 --namespace kube-system  --set nodeSelectorDaemonSet.nodeinfo=gpu
# helm  list --all-namespaces
# helm uninstall xxx --namespace kube-system
```

5. Delete chart repo
```bash
# helm repo remove myrepo
```
