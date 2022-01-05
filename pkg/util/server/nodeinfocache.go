package server

import (
	"fmt"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"
	"sync"
)

func NewNodeGpuInfoMap() *NodeGpuInfoMap {
	return &NodeGpuInfoMap{
		nodegpuinfo:     make(map[string]*NodeGpuInfo),
		nodeDeviceInUse: make(map[string]sets.String),
	}
}

// struct to store gpu info from all nodes
type NodeGpuInfoMap struct {
	sync.RWMutex
	nodegpuinfo     map[string]*NodeGpuInfo
	nodeDeviceInUse map[string]sets.String
}

func (gim *NodeGpuInfoMap) GetNodeGpuInfo(node string) (value *NodeGpuInfo, exist bool) {
	gim.RLock()
	defer gim.RUnlock()
	value, exist = gim.nodegpuinfo[node]
	return
}

func (gim *NodeGpuInfoMap) SetNodeGpuInfo(node string, value *NodeGpuInfo) {
	gim.Lock()
	defer gim.Unlock()
	gim.nodegpuinfo[node] = value
}

func (gim *NodeGpuInfoMap) GetAllNodeGpuInfo() []*NodeGpuInfo {
	gim.RLock()
	defer gim.RUnlock()
	ngiList := make([]*NodeGpuInfo, 0, len(gim.nodegpuinfo))
	for _, v := range gim.nodegpuinfo {
		ngiList = append(ngiList, v)
	}
	return ngiList
}

func (gim *NodeGpuInfoMap) DumpNodeGpuInfo() string {
	gim.RLock()
	defer gim.RUnlock()
	sb := strings.Builder{}
	for k, v := range gim.nodegpuinfo {
		sb.WriteString(fmt.Sprintf("node[%s]:", k))
		for _, gi := range v.GpuInfos {
			sb.WriteString(fmt.Sprintf("%#v", *gi))
		}
	}
	return sb.String()
}

func (gim *NodeGpuInfoMap) SetNodeDeviceInUse(node string, deviceInUse sets.String) {
	gim.Lock()
	defer gim.Unlock()
	if gim.nodeDeviceInUse[node] == nil {
		gim.nodeDeviceInUse[node] = deviceInUse
		return
	}

	gim.nodeDeviceInUse[node].Insert(deviceInUse.UnsortedList()...)
	did2delete := gim.nodeDeviceInUse[node].Difference(deviceInUse)
	gim.nodeDeviceInUse[node].Delete(did2delete.UnsortedList()...)
}

func (gim *NodeGpuInfoMap) GetNodeDeviceInUse(node string) (value sets.String) {
	gim.Lock()
	defer gim.Unlock()
	if gim.nodeDeviceInUse[node] == nil {
		gim.nodeDeviceInUse[node] = sets.NewString()
	}
	value = gim.nodeDeviceInUse[node]
	return
}
