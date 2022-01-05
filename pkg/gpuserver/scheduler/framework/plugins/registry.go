package plugins

import (
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/plugins/names"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/plugins/noderesources"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/runtime"
)

// NewInTreeRegistry builds the registry with all the in-tree plugins.
func NewInTreeRegistry() runtime.Registry {
	return runtime.Registry{
		names.GpuModelFitName: noderesources.NewGpuModelFit,
	}
}
