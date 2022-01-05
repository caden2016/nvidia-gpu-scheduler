package controller

import (
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller/controllertype"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/nodeinfo"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/plugins"
	fwruntime "github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework/runtime"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"reflect"
	"strings"
)

var (

	// podresourcesIndex store pod uid(namespace/name) with podresources.
	podresourcesIndex = make(map[string]*PodResourcesDetail)
	// map node name to set of the pod uid(namespace/name)
	node2podset      = make(map[string]sets.String)
	podresourcesChan = make(chan *PodResourceUpdate, 1000)
	chanGetSignal    = make(chan struct{})
	chanGetResult    = make(chan []*PodResourcesDetail)

	chanWatchIndex  = make(map[string]chan *PodResourceUpdate)
	chanWatchSignal = make(chan *controllertype.WatcherInfo, 100)
	chanWatchExit   = make(chan string, 100)

	// nodegpuinfomap store node name with gpu model info.
	nodegpuinfomap = serverutil.NewNodeGpuInfoMap()
	/*podresourcesapi.PodResources looks linke
	{
		"name": "gpu-new1-f7d6566bb-5qxbj",
		"namespace": "kube-system",
		"containers": [
			{
				"name": "gpu-new1",
				"devices": [
					{
						"resource_name": "nvidia.com/gpu",
						"device_ids": [
							"GPU-ef278bb8-6ee4-4758-0a14-040d90b6536e"
						]
					}
				]
			}
		]
	}
	*/
)

// ServerController is the main controller to process api requests.
// Index the pod gpu usage info with podresourcesIndex.
// Index the node gpu info with nodegpuinfomap.
type ServerController struct {
	stop        <-chan struct{}
	NHC         *NodeHealthChecker
	FW          framework.Framework
	FWNI        *nodeinfo.NodeInfo
	parallelism int
}

func (sc *ServerController) SendToStore(pru *PodResourceUpdate) {
	podresourcesChan <- pru
}

func (sc *ServerController) GetFromStore() <-chan []*PodResourcesDetail {
	klog.V(1).Infof("len(podresourcesChan):%d", len(podresourcesChan))
	chanGetSignal <- struct{}{}
	return chanGetResult
}

func (sc *ServerController) AddWatcher(chprd chan *PodResourceUpdate) string {
	watch_uuid := string(uuid.NewUUID())
	wi := &controllertype.WatcherInfo{ChanToAdd: chprd, WatchUUID: watch_uuid}
	chanWatchSignal <- wi
	return watch_uuid
}

func (sc *ServerController) DelWatcher(watch_uuid string) {
	chanWatchExit <- watch_uuid
}

func (sc *ServerController) GetNodeGpuInfoMap() *serverutil.NodeGpuInfoMap {
	return nodegpuinfomap
}

func (sc *ServerController) GetParallelism() int {
	if sc.parallelism > 0 {
		return sc.parallelism
	}
	return options.SchedulerRouter_Parallelism_Default
}

func NewServerController(nhc *NodeHealthChecker, parallelism int, stop <-chan struct{}) (*ServerController, error) {
	registryInTree := plugins.NewInTreeRegistry()
	fw, err := fwruntime.NewFramework(registryInTree)
	if err != nil {
		return nil, err
	}
	ni := nodeinfo.NewNodeInfo(nodegpuinfomap, nhc.GetNodeStatusIndex())
	sc := &ServerController{NHC: nhc, FW: fw, FWNI: ni, parallelism: parallelism, stop: stop}
	go mainProcessLoop(stop)
	return sc, nil
}

// MainProcessLoop process signal or stopChan, podresources changed, list or watch podresoueces
func mainProcessLoop(stopCh <-chan struct{}) {
	klog.Infof("ServerController MainProcessLoop start")

	result := make([]*PodResourcesDetail, 0, len(podresourcesIndex))
PROCESSLOOP:
	for {
		select {
		case <-stopCh:
			break PROCESSLOOP

		case pru := <-podresourcesChan:
			podresourcesIndexChanged := false

			watchPodResourceUpdate := &PodResourceUpdate{
				PodResourcesSYNC: make([]*PodResourcesDetail, 0, len(pru.PodResourcesSYNC)),
				PodResourcesDEL:  pru.PodResourcesDEL,
				NodeName:         pru.NodeName,
			}

			// Update nodeDeviceInUse of each node, used in schedulerRouter.podFitOnNode
			deviceBusySet := serverutil.GetBusyDeviceSet(pru.PodResourcesSYNC)
			nodegpuinfomap.SetNodeDeviceInUse(pru.NodeName, deviceBusySet)

			if node2podset[pru.NodeName] == nil {
				node2podset[pru.NodeName] = sets.NewString()
			}
			podset := sets.NewString()
			for _, prd := range pru.PodResourcesSYNC {
				podidx := strings.Join([]string{prd.Namespace, prd.Name}, "/")
				oldprd := podresourcesIndex[podidx]
				podset.Insert(podidx)
				node2podset[pru.NodeName].Insert(podidx)
				var oldpr *podresourcesapi.PodResources
				if oldprd != nil {
					oldpr = oldprd.PodResources
				}
				podresourcesIndex[podidx] = prd
				if !reflect.DeepEqual(oldpr, prd.PodResources) {
					podresourcesIndexChanged = true
					watchPodResourceUpdate.PodResourcesSYNC = append(watchPodResourceUpdate.PodResourcesSYNC, prd)
				}
			}
			// Case:123001 - fix
			podset2delete := node2podset[pru.NodeName].Difference(podset)
			klog.V(3).Infof("podset2delete:%v", podset2delete.UnsortedList())
			for _, k := range podset2delete.UnsortedList() {
				delete(podresourcesIndex, k)
				node2podset[pru.NodeName].Delete(k)
				podresourcesIndexChanged = true
				if len(pru.PodResourcesDEL) == 0 {
					nsname := strings.SplitN(k, "/", 2)
					watchPodResourceUpdate.PodResourcesDEL = append(watchPodResourceUpdate.PodResourcesDEL,
						&podresourcesapi.PodResources{Namespace: nsname[0], Name: nsname[1]})
				}
			}

			for _, pr := range pru.PodResourcesDEL {
				podidx := strings.Join([]string{pr.Namespace, pr.Name}, "/")
				delete(podresourcesIndex, podidx)
				node2podset[pru.NodeName].Delete(podidx)
				podresourcesIndexChanged = true
			}

			result = make([]*PodResourcesDetail, 0, len(podresourcesIndex))
			for pk := range podresourcesIndex {
				result = append(result, podresourcesIndex[pk])
			}

			// If podresources changed notice all watchers
			if podresourcesIndexChanged {
				klog.V(4).Infof("podresourcesIndexLast differ from result, so notice watch")
				for _, cw := range chanWatchIndex {
					select {
					case cw <- watchPodResourceUpdate:
					default:
					}
				}
			}

		case <-chanGetSignal:
			klog.V(1).Infof("request for podresources arrives, prepare data and send back")
			// If request for podresources arrives, prepare data and send back
			chanGetResult <- result

		case watchinfo := <-chanWatchSignal:
			chanWatchIndex[watchinfo.WatchUUID] = watchinfo.ChanToAdd

		case watch_uuid := <-chanWatchExit:
			delete(chanWatchIndex, watch_uuid)
		}
	}

	klog.Infof("MainProcessLoop clean all watch connections")
	for _, cw := range chanWatchIndex {
		close(cw)
	}

	klog.Infof("MainProcessLoop end")
}
