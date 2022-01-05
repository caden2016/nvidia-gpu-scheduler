# helm-chart

## helm的chart仓库地址为：https://caden2016.github.io/nvidia-gpu-scheduler

## Chart仓库的使用方法

1. 添加chart仓库
``` 
# helm repo add myrepo https://caden2016.github.io/nvidia-gpu-scheduler
# helm repo update
```

2. 添加成功
```
# helm repo list
NAME  	URL                                   
myrepo	https://caden2016.github.io/nvidia-gpu-scheduler
```

3. 搜索chart包
```
# helm search repo myrepo
NAME                              	CHART VERSION	APP VERSION	DESCRIPTION                                            
myrepo/nvidia-gpu-scheduler       	0.1.0        	0.1.0      	A Helm chart for nvidia-gpu-scheduler on Kubernetes 
```

4. 安装chart包
```
# helm install xxx myrepo/nvidia-gpu-scheduler --version 0.1.0 --namespace kube-system  --set nodeSelectorDaemonSet.nodeinfo=gpu
```

xxx为relaese名字, nodeinfo=gpu为gpuserver-ds部署到gpu节点上的label
