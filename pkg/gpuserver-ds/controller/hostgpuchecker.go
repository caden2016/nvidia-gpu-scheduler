package controller

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	serverdsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/serverds"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"reflect"
	"time"
)

func NewHostGpuInfoChecker(checkInterval time.Duration, stop <-chan struct{}) (*HostGpuInfoChecker, error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to initialize NVML: %v", nvml.ErrorString(ret))
	}

	return &HostGpuInfoChecker{
		checkInterval: checkInterval,
		stop:          stop,
		gpuinfoChan:   make(chan *NodeGpuInfo),
		modelSetLast:  make(map[string]sets.String),
	}, nil

}

// HostGpuInfoChecker check node gpu model and populate it.
// It depends on the package github.com/NVIDIA/go-nvml/pkg/nvml
// It Start to populate gpu model info each interval and never stop until stop chan signal.
type HostGpuInfoChecker struct {
	checkInterval time.Duration
	stop          <-chan struct{}
	gpuinfoChan   chan *NodeGpuInfo
	// map the device model to the last observed device ids in set
	modelSetLast map[string]sets.String
}

func (gic *HostGpuInfoChecker) Start() error {
	go func() {
		defer func() {
			ret := nvml.Shutdown()
			if ret != nvml.SUCCESS {
				klog.Errorf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
			}
		}()

		klog.Infof("HostGpuInfoChecker started with check interval:%v", gic.checkInterval)
		ct := time.Tick(gic.checkInterval)
	CKECKLOOP:
		for {
			select {
			case <-ct:
				count, ret := nvml.DeviceGetCount()
				if ret != nvml.SUCCESS {
					klog.Errorf("Unable to get device count: %v", nvml.ErrorString(ret))
					continue
				}
				needNotify := true
				nodegpuinfo := &NodeGpuInfo{GpuInfos: make(map[string]*GpuInfo), Models: make(map[string]sets.String)}

				for i := 0; i < count; i++ {
					device, ret := nvml.DeviceGetHandleByIndex(i)
					if ret == nvml.SUCCESS {
						did, ret := device.GetUUID()
						if ret == nvml.SUCCESS {
							gpuinfo, err := updateGpuInfo(did)
							if err != nil {
								klog.Errorf("updateGpuInfo: %v", err)
								needNotify = false
								break
							}
							nodegpuinfo.GpuInfos[did] = gpuinfo
							nmodel := util.NormalizeModelName(gpuinfo.Model)
							if nodegpuinfo.Models[nmodel] == nil {
								nodegpuinfo.Models[nmodel] = sets.NewString()
							}
							nodegpuinfo.Models[nmodel].Insert(did)
						} else {
							klog.Errorf("Unable to get device GetName at index %d: %v", i, nvml.ErrorString(ret))
							needNotify = false
							break
						}
					} else {
						klog.Errorf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
						klog.Infof("Try to initiate nvml again")
						// reinit to restore
						ret := nvml.Init()
						if ret != nvml.SUCCESS {
							klog.Errorf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
						}
						needNotify = false
						break
					}
				}

				if needNotify {
					if !reflect.DeepEqual(gic.modelSetLast, nodegpuinfo.Models) {
						klog.Infof("Notify the node devices uuid changed: original:%s current:%s",
							serverdsutil.DumpModelSetInfo(gic.modelSetLast), serverdsutil.DumpModelSetInfo(nodegpuinfo.Models))
						gic.modelSetLast = nodegpuinfo.Models
						gic.gpuinfoChan <- nodegpuinfo
					}
				}

			case <-gic.stop:
				break CKECKLOOP
			}
		}
		klog.Infof("HostGpuInfoChecker stopped")
	}()

	return nil
}

func (gic *HostGpuInfoChecker) GetGpuInfoChan() <-chan *NodeGpuInfo {
	return gic.gpuinfoChan
}
