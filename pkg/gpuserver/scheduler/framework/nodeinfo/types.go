package nodeinfo

import (
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
)

func NewNodeInfo(ngim *serverutil.NodeGpuInfoMap, nsm *serverutil.NodeStatusMap) *NodeInfo {
	return &NodeInfo{
		NGIM: ngim,
		NSM:  nsm,
	}
}

// NodeInfo knows the gpu info of each node.
// Help to decide which node is permitted or not.
type NodeInfo struct {
	NGIM *serverutil.NodeGpuInfoMap
	NSM  *serverutil.NodeStatusMap
}
