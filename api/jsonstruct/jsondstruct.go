package jsonstruct

import (
	"k8s.io/apimachinery/pkg/util/sets"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"time"
)

type PodResourceUpdate struct {
	PodResourcesSYNC []*PodResourcesDetail //we don't have to tell the add and update
	PodResourcesDEL  []*podresourcesapi.PodResources
	NodeName         string
}

type GpuInfo struct {
	DeviceId string `json:"device_id,omitempty"`
	Brand    string `json:"device_brand,omitempty"`
	Model    string `json:"device_model,omitempty"`
	BusId    string `json:"device_busid,omitempty"`
	NodeName string `json:"device_node,omitempty"`
}

type ContainerResourcesDetail struct {
	Name       string     `json:"container_name,omitempty"`
	DeviceInfo []*GpuInfo `json:"device_info,omitempty"`
}

type PodResourcesDetail struct {
	*podresourcesapi.PodResources
	ContainerDevices *[]*ContainerResourcesDetail `json:"containers_device,omitempty"`
}

type NodeGpuInfo struct {
	NodeName string                 `json:"device_node,omitempty"`
	GpuInfos map[string]*GpuInfo    `json:"device_infos,omitempty"`
	Models   map[string]sets.String `json:"device_models,omitempty"`
	// Used in gpuserver to record the time message received by the gpuserver
	ReportTime time.Time `json:"report_time,omitempty"`
}
