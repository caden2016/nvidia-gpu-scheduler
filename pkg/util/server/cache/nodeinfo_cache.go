package cache

import (
	"fmt"
	"strings"
	"sync"

	resourcesschedulerv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

var DefaultGpuNodeCache = NewGpuNodeCache()

func NewGpuNodeCache() *GpuNodeCache {
	return &GpuNodeCache{
		gpuNodeMap: make(map[string]*resourcesschedulerv1.GpuNode),
	}
}

// GpuNodeCache to store gpu info from all nodes.
type GpuNodeCache struct {
	sync.RWMutex
	gpuNodeMap map[string]*resourcesschedulerv1.GpuNode
}

func (gnc *GpuNodeCache) SetGpuNode(node string, value *resourcesschedulerv1.GpuNode) {
	gnc.Lock()
	defer gnc.Unlock()
	gnc.gpuNodeMap[node] = value
}

func (gnc *GpuNodeCache) DumpNodeGpuInfo() string {
	gnc.RLock()
	defer gnc.RUnlock()
	sb := strings.Builder{}
	for k, v := range gnc.gpuNodeMap {
		sb.WriteString(fmt.Sprintf("node[%s]:", k))
		for _, gi := range v.Spec.GpuInfos {
			sb.WriteString(fmt.Sprintf("%#v;", *gi))
		}
	}
	return sb.String()
}

func (gnc *GpuNodeCache) GetDeviceInUse(node string) (value sets.String) {
	gnc.Lock()
	defer gnc.Unlock()
	value = sets.NewString()
	if gnc.gpuNodeMap[node] == nil {
		return
	}
	value.Insert(gnc.gpuNodeMap[node].Spec.NodeDeviceInUse...)
	return
}

// GetFreeDeviceByModel gets the free gpu device set by model type.
func (gnc *GpuNodeCache) GetFreeDeviceByModel(node, model string) (value sets.String) {
	gnc.RLock()
	defer gnc.RUnlock()
	value = sets.NewString()
	if gnc.gpuNodeMap[node] == nil {
		return
	}
	modelList := gnc.gpuNodeMap[node].Spec.Models[model]
	if len(modelList) == 0 {
		return
	}
	value.Insert(modelList...)
	value.Delete(gnc.gpuNodeMap[node].Spec.NodeDeviceInUse...)
	return
}

func (gnc *GpuNodeCache) CheckNodeHealth(node string) (exist, health bool) {
	gnc.RLock()
	defer gnc.RUnlock()
	if gnc.gpuNodeMap[node] == nil {
		return
	}

	exist = true
	if gnc.gpuNodeMap[node].Status.Health == resourcesschedulerv1.StatusHealth {
		health = true
	}
	return
}
