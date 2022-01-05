package apis

import (
	"github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
)

func NewPodResource(spec *PodResourceSpec, status *PodResourceStatus) *PodResource {
	return &PodResource{
		TypeMeta: TypeMeta{Kind: options.KIND, APIVersion: options.APIVERSION},
		Spec:     spec,
		Status:   status,
	}
}

type PodResource struct {
	TypeMeta
	Spec   *PodResourceSpec   `json:"spec,omitempty"`
	Status *PodResourceStatus `json:"status,omitempty"`
}

type PodResourceSpec struct {
	Name             string                                  `json:"pod_name,omitempty"`
	Namespace        string                                  `json:"pod_namespace,omitempty"`
	ContainerDevices *[]*jsonstruct.ContainerResourcesDetail `json:"containers_device,omitempty"`
}

type PodResourceStatus struct {
	LastChangedTime string `json:"last_changed_time,omitempty"`
}
