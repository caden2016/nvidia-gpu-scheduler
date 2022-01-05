package server

import (
	"sync"
	"time"
)

func NewNodeStatusMap() *NodeStatusMap {
	return &NodeStatusMap{
		nodestatusinfo: make(map[string]*NodeStatus),
	}
}

// struct to store node status info from NodeHealthChecker
type NodeStatusMap struct {
	sync.RWMutex
	nodestatusinfo map[string]*NodeStatus
}

type NodeStatus struct {
	Name            string
	Health          bool
	LastHealthyTime time.Time
}

func (gim *NodeStatusMap) GetNodeStatus(node string) (value *NodeStatus, exist bool) {
	gim.RLock()
	defer gim.RUnlock()
	value, exist = gim.nodestatusinfo[node]
	return
}

func (gim *NodeStatusMap) SetNodeGpuInfo(node string, value *NodeStatus) {
	gim.Lock()
	defer gim.Unlock()
	gim.nodestatusinfo[node] = value
}

func (gim *NodeStatusMap) ListNodeStatus() []*NodeStatus {
	gim.RLock()
	defer gim.RUnlock()
	r := make([]*NodeStatus, 0, len(gim.nodestatusinfo))
	for _, v := range gim.nodestatusinfo {
		r = append(r, v)
	}
	return r
}
