package apis

import (
	"github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"time"
)

func NewGpuInfo(spec *GpuInfoSpec, status *GpuInfoStatus) *GpuInfo {
	return &GpuInfo{
		TypeMeta: TypeMeta{Kind: options.GPUKIND, APIVersion: options.APIVERSION},
		Spec:     spec,
		Status:   status,
	}
}

type GpuInfo struct {
	TypeMeta
	Spec   *GpuInfoSpec   `json:"spec,omitempty"`
	Status *GpuInfoStatus `json:"status,omitempty"`
}

type GpuInfoSpec struct {
	NodeName string                         `json:"device_node,omitempty"`
	GpuInfos map[string]*jsonstruct.GpuInfo `json:"device_infos,omitempty"`
	Models   map[string][]string            `json:"device_models,omitempty"`
	//record the time gpuinfo received by the gpuserver
	ReportTime      time.Time `json:"report_time,omitempty"`
	NodeDeviceInUse []string  `json:"device_busy"`
}

type GpuInfoStatus struct {
	Health          string `json:"node_health,omitempty"`
	LastHealthyTime string `json:"node_last_health_time,omitempty"`
}
