package controller

import (
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/plugins"
	fwruntime "github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServerController is the main controller to process api requests.
// Index the pod gpu usage info with podresourcesIndex.
// Index the node gpu info with nodegpuinfomap.
type ServerController struct {
	stop         <-chan struct{}
	FW           framework.Framework
	parallelism  int
	GpuMgrClient client.Client
}

func (sc *ServerController) GetParallelism() int {
	if sc.parallelism > 0 {
		return sc.parallelism
	}
	return options.SchedulerRouter_Parallelism_Default
}

func NewServerController(stop <-chan struct{}, parallelism int, gpuMgrClient client.Client) (*ServerController, error) {
	registryInTree := plugins.NewInTreeRegistry()
	fw, err := fwruntime.NewFramework(registryInTree)
	if err != nil {
		return nil, err
	}
	sc := &ServerController{FW: fw, parallelism: parallelism, stop: stop, GpuMgrClient: gpuMgrClient}
	return sc, nil
}
