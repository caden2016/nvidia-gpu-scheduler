package server

import (
	"github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"time"
)

func GetPodRequestGpuNum(pod *corev1.Pod) int64 {
	var numLimit int64
	for _, c := range pod.Spec.Containers {
		if c.Resources.Limits != nil {
			if gpulimit, exist := c.Resources.Limits[options.NVIDIAGPUResourceName]; exist {
				if !gpulimit.IsZero() {
					numLimit += gpulimit.Value()
				}
			}
		}
	}
	return numLimit
}

func GetBusyDeviceSet(prl []*jsonstruct.PodResourcesDetail) sets.String {
	deviceBusySet := sets.NewString()
	for _, prd := range prl {
		for _, cr := range prd.PodResources.Containers {
			for _, device := range cr.Devices {
				for _, did := range device.DeviceIds {
					deviceBusySet.Insert(did)
				}
			}
		}
	}
	return deviceBusySet
}

func MapSetToList(mapset map[string]sets.String) map[string][]string {
	r := make(map[string][]string)
	for k, v := range mapset {
		r[k] = v.List()
	}
	return r
}

func NodeStatusToGpuInfoStatus(status *NodeStatus) *apis.GpuInfoStatus {
	r := &apis.GpuInfoStatus{Health: "False"}
	if status == nil {
		r.Health = "UnKnow"
		r.LastHealthyTime = "UnKnow"
		return r
	}

	if status.Health {
		r.Health = "True"
	}
	r.LastHealthyTime = status.LastHealthyTime.Local().Format(time.RFC3339)
	return r
}

func PodResourcesDetailToPodResource(prdList []*jsonstruct.PodResourcesDetail) []*apis.PodResource {
	r := make([]*apis.PodResource, 0, len(prdList))
	for _, prd := range prdList {
		r = append(r, apis.NewPodResource(
			&apis.PodResourceSpec{
				Name:             prd.Name,
				Namespace:        prd.Namespace,
				ContainerDevices: prd.ContainerDevices,
			},
			&apis.PodResourceStatus{
				LastChangedTime: time.Now().Format(time.RFC3339),
			}))
	}
	return r
}
