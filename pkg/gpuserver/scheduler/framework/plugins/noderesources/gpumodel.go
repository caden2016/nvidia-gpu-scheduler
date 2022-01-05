package noderesources

import (
	"context"
	"fmt"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/nodeinfo"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/plugins/names"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const GpuModelFitName = names.GpuModelFitName

var _ framework.FilterPlugin = &GpuModelFit{}

func NewGpuModelFit() (framework.Plugin, error) {
	return &GpuModelFit{}, nil
}

// GpuModelFit is a plugin that checks if a node has sufficient gpu with model requested.
type GpuModelFit struct {
}

func (f *GpuModelFit) Name() string {
	return GpuModelFitName
}

func (f *GpuModelFit) Filter(ctx context.Context, pod *corev1.Pod, nodeInfo *nodeinfo.NodeInfo, node string) (status *framework.Status) {
	status = &framework.Status{Accepted: true}
	if len(pod.Annotations) != 0 {
		if reqModel, exist := pod.Annotations[options.SCHEDULE_ANNOTATION]; exist {
			reqModel = util.NormalizeModelName(reqModel)
			nodegpuinfo, nexist := nodeInfo.NGIM.GetNodeGpuInfo(node)
			nodestatus, _ := nodeInfo.NSM.GetNodeStatus(node)
			if nodestatus == nil {
				status.Err = fmt.Errorf("nodeName:%s is not exist.", node)
				status.Accepted = false
				return
			} else if !nodestatus.Health {
				status.Err = fmt.Errorf("nodeName:%s is not health. nodestatus.LastHealthyTime:%v", node, nodestatus.LastHealthyTime.Local())
				status.Accepted = false
				return
			}
			if !nexist {
				status.Err = fmt.Errorf("nodeName:%s not exist. nodegpuinfomap:%s", node, nodeInfo.NGIM.DumpNodeGpuInfo())
				status.Accepted = false
				return
			}

			if nodegpuinfo.Models[reqModel] != nil {
				nodeDeviceInUse := nodeInfo.NGIM.GetNodeDeviceInUse(node)
				availDevice := nodegpuinfo.Models[reqModel].Difference(nodeDeviceInUse)
				reqDeviceNum := serverutil.GetPodRequestGpuNum(pod)
				klog.Infof("node:[%s] pod[%s/%s] reqDeviceNum:%d ,availDevice:%v",
					node, pod.Namespace, pod.Name, reqDeviceNum, availDevice.List())
				if reqDeviceNum > int64(availDevice.Len()) {
					status.Err = fmt.Errorf("node:[%s] pod[%s/%s] reqGpuNum:%d > availNum:%d",
						node, pod.Namespace, pod.Name, reqDeviceNum, availDevice.Len())
					status.Accepted = false
				}
			} else {
				status.Err = fmt.Errorf("node:[%s] pod[%s/%s] reqModel:%s not exist in nodegpuinfo.Models %v",
					node, pod.Namespace, pod.Name, reqModel, nodegpuinfo.Models)
				status.Accepted = false
			}
		}
	}
	return
}

func (f *GpuModelFit) Score(ctx context.Context, pod *corev1.Pod, nodeInfo *nodeinfo.NodeInfo, node string) (score int64, status *framework.Status) {
	status = &framework.Status{Accepted: true}
	if len(pod.Annotations) != 0 {
		if reqModel, exist := pod.Annotations[options.SCHEDULE_ANNOTATION]; exist {
			reqModel = util.NormalizeModelName(reqModel)
			nodegpuinfo, nexist := nodeInfo.NGIM.GetNodeGpuInfo(node)
			nodestatus, _ := nodeInfo.NSM.GetNodeStatus(node)
			if nodestatus == nil {
				status.Err = fmt.Errorf("nodeName:%s is not exist.", node)
				status.Accepted = false
				return
			} else if !nodestatus.Health {
				status.Err = fmt.Errorf("nodeName:%s is not health. nodestatus.LastHealthyTime:%v", node, nodestatus.LastHealthyTime.Local())
				status.Accepted = false
				return
			}
			if !nexist {
				status.Err = fmt.Errorf("nodeName:%s not exist. nodegpuinfomap:%s", node, nodeInfo.NGIM.DumpNodeGpuInfo())
				status.Accepted = false
				return
			}

			if nodegpuinfo.Models[reqModel] != nil {
				nodeDeviceInUse := nodeInfo.NGIM.GetNodeDeviceInUse(node)
				availDevice := nodegpuinfo.Models[reqModel].Difference(nodeDeviceInUse)
				//Set score to be the num of the available gpus with model that pod requested.
				score = int64(availDevice.Len())
			} else {
				status.Err = fmt.Errorf("node:[%s] pod[%s/%s] reqModel:%s not exist in nodegpuinfo.Models %v",
					node, pod.Namespace, pod.Name, reqModel, nodegpuinfo.Models)
				status.Accepted = false
			}
		}
	}
	return
}
