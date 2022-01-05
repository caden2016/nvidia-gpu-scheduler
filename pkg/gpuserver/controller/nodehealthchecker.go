package controller

import (
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"k8s.io/klog"
	"time"
)

func NewNodeHealthChecker(checkInterval time.Duration, nodeStatusTTL time.Duration, stop <-chan struct{}) *NodeHealthChecker {
	return &NodeHealthChecker{
		checkInterval:   checkInterval,
		nodeStatusTTL:   nodeStatusTTL,
		stop:            stop,
		nodeStatusIndex: serverutil.NewNodeStatusMap(),
		notifyHealth:    make(chan string),
	}
}

// NodeHealthChecker checks node status.
// It Start check each interval and never stop until stop chan signal.
// According to the http check from the gpuserver-ds /health?node=nodename.
// If the gpuserver-ds not check in a period of time (NodeHealthChecker_NodeTTL) then mark it unhealthy.
type NodeHealthChecker struct {
	checkInterval   time.Duration
	nodeStatusTTL   time.Duration
	stop            <-chan struct{}
	nodeStatusIndex *serverutil.NodeStatusMap
	notifyHealth    chan string
}

func (nhc *NodeHealthChecker) Start() error {
	go func() {
		klog.Infof("NodeHealthChecker started with check interval:%v", nhc.checkInterval)
		ct := time.Tick(nhc.checkInterval)
	CKECKLOOP:
		for {
			select {
			case <-ct:
				nhc.updateNodeStatusIndexCheck()
			case node := <-nhc.notifyHealth:
				nhc.updateNodeStatusIndexHealth(node)
			case <-nhc.stop:
				break CKECKLOOP
			}
		}
		klog.Infof("NodeHealthChecker stopped")
	}()
	return nil
}

func (nhc *NodeHealthChecker) NotifyHealth(node string) {
	nhc.notifyHealth <- node
}

func (nhc *NodeHealthChecker) GetNodeStatus(node string) *serverutil.NodeStatus {
	r, _ := nhc.nodeStatusIndex.GetNodeStatus(node)
	return r
}

func (nhc *NodeHealthChecker) updateNodeStatusIndexHealth(node string) {
	nodeStatus, exist := nhc.nodeStatusIndex.GetNodeStatus(node)
	if !exist {
		nhc.nodeStatusIndex.SetNodeGpuInfo(node, &serverutil.NodeStatus{Name: node, Health: true, LastHealthyTime: time.Now()})
		return
	}
	nodeStatus.LastHealthyTime = time.Now()
	if !nodeStatus.Health {
		klog.Infof("node:%s LastHealthyTime:%v ,set from unhealthy to healthy.", nodeStatus.Name, nodeStatus.LastHealthyTime)
	}
	nodeStatus.Health = true
}

func (nhc *NodeHealthChecker) updateNodeStatusIndexCheck() {
	for _, nodeStatus := range nhc.nodeStatusIndex.ListNodeStatus() {
		if time.Now().After(nodeStatus.LastHealthyTime.Add(nhc.nodeStatusTTL)) {
			if nodeStatus.Health {
				nodeStatus.Health = false
				klog.Infof("node:%s LastHealthyTime:%v ,set from healthy to unhealthy.", nodeStatus.Name, nodeStatus.LastHealthyTime)
			}
		}
	}
}

func (nhc *NodeHealthChecker) GetNodeStatusIndex() *serverutil.NodeStatusMap {
	return nhc.nodeStatusIndex
}
