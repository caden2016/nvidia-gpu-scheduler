package noderesources

import (
	"context"
	"fmt"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/plugins/names"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server/cache"
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

func (f *GpuModelFit) Filter(ctx context.Context, pod *corev1.Pod, node string) (status *framework.Status) {
	status = &framework.Status{Accepted: true}
	if len(pod.Annotations) != 0 {
		if reqModel, exist := pod.Annotations[options.SCHEDULE_ANNOTATION]; exist {
			reqModel = util.NormalizeModelName(reqModel)
			nexist, nhealth := cache.DefaultGpuNodeCache.CheckNodeHealth(node)
			if !nexist {
				status.Err = fmt.Errorf("nodeName:%s not exist. nodeCache:%s", node, cache.DefaultGpuNodeCache.DumpNodeGpuInfo())
				status.Accepted = false
				return
			} else if !nhealth {
				status.Err = fmt.Errorf("nodeName:%s is not health", node)
				status.Accepted = false
				return
			}

			freeDevice := cache.DefaultGpuNodeCache.GetFreeDeviceByModel(node, reqModel)
			if freeDevice.Len() != 0 {
				reqDeviceNum := serverutil.GetPodRequestGpuNum(pod)
				klog.Infof("node:[%s] pod[%s/%s] reqDeviceNum:%d ,availDevice:%v",
					node, pod.Namespace, pod.Name, reqDeviceNum, freeDevice.List())
				if reqDeviceNum > int64(freeDevice.Len()) {
					status.Err = fmt.Errorf("node:[%s] pod[%s/%s] reqGpuNum:%d > availNum:%d",
						node, pod.Namespace, pod.Name, reqDeviceNum, freeDevice.Len())
					status.Accepted = false
				}
			} else {
				status.Err = fmt.Errorf("node:[%s] pod[%s/%s] reqModel:%s not exist",
					node, pod.Namespace, pod.Name, reqModel)
				status.Accepted = false
			}
		}
	}
	return
}

func (f *GpuModelFit) Score(ctx context.Context, pod *corev1.Pod, node string) (score int64, status *framework.Status) {
	status = &framework.Status{Accepted: true}
	if len(pod.Annotations) != 0 {
		if reqModel, exist := pod.Annotations[options.SCHEDULE_ANNOTATION]; exist {
			reqModel = util.NormalizeModelName(reqModel)
			reqModel = util.NormalizeModelName(reqModel)
			nexist, nhealth := cache.DefaultGpuNodeCache.CheckNodeHealth(node)
			if !nexist {
				status.Err = fmt.Errorf("nodeName:%s not exist. nodeCache:%s", node, cache.DefaultGpuNodeCache.DumpNodeGpuInfo())
				status.Accepted = false
				return
			} else if !nhealth {
				status.Err = fmt.Errorf("nodeName:%s is not health", node)
				status.Accepted = false
				return
			}

			freeDevice := cache.DefaultGpuNodeCache.GetFreeDeviceByModel(node, reqModel)
			if freeDevice.Len() != 0 {
				//Set score to be the num of the available gpus with model that pod requested.
				score = int64(freeDevice.Len())
			} else {
				status.Err = fmt.Errorf("node:[%s] pod[%s/%s] reqModel:%s not exist",
					node, pod.Namespace, pod.Name, reqModel)
				status.Accepted = false
			}
		}
	}
	return
}
