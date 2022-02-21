package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"

	gpunodev1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	gpupodv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	gpuclientset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpunode/clientset/versioned"
	gpupodcleintset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpupod/clientset/versioned"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver-ds/podresources"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/info/metadata"
	serverdsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/serverds"
	"google.golang.org/grpc"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
)

var (
	brand2type = [18]string{"BRAND_UNKNOWN", "BRAND_QUADRO", "BRAND_TESLA", "BRAND_NVS", "BRAND_GRID", "BRAND_GEFORCE",
		"BRAND_TITAN", "BRAND_NVIDIA_VAPPS", "BRAND_NVIDIA_VPC", "BRAND_NVIDIA_VCS", "BRAND_NVIDIA_VWS", "BRAND_NVIDIA_VGAMING",
		"BRAND_QUADRO_RTX", "BRAND_NVIDIA_RTX", "BRAND_NVIDIA", "BRAND_GEFORCE_RTX", "BRAND_TITAN_RTX", "BRAND_COUNT"}
	ttlCacheGpu = serverdsutil.NewTTLCacheGpu(5 * time.Second)

	updateBackoff = wait.Backoff{
		Steps:    options.PRODUCE_MAX_ERRORCOUNT,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   1.0,
	}
)

func NewServerDSController(stop <-chan struct{}, goonChan <-chan struct{}, removeChan <-chan *PodResourceUpdate, gpuinfoChan <-chan *NodeGpuInfo, podresourcesep string, gpuClient gpuclientset.Interface, gpuPodClient gpupodcleintset.Interface) (*ServerDSController, error) {
	nodeName := os.Getenv("NODENAME")
	if nodeName == "" {
		return nil, fmt.Errorf("unable get env NODENAME")
	}

	client, conn, err := podresources.GetClient(podresourcesep, options.DefaultPodResourcesTimeoutConnect, options.DefaultPodResourcesMaxSize)
	if err != nil {
		return nil, fmt.Errorf("error getting grpc client: %v", err)
	}

	dsc := &ServerDSController{
		goonChan:     goonChan,
		removeChan:   removeChan,
		stop:         stop,
		gpuinfoChan:  gpuinfoChan,
		nodeName:     nodeName,
		prclient:     client,
		grpconn:      conn,
		svcName:      metadata.ServiceName(),
		gpuClient:    gpuClient,
		gpuPodClient: gpuPodClient,
		gpuPodLast:   make(map[string]*gpupodv1.GpuPod),
	}

	return dsc, nil
}

// ServerDSController is the main controller to signal gpu usage info to the server.
// Signal pod gpu usage info to the server.
// Signal node gpu info to the server.
type ServerDSController struct {
	stop <-chan struct{}
	// Signal the server the gpu pod on the node is sync. which means the gpu info changed.
	goonChan <-chan struct{}
	// Signal the server the gpu pod on the node is deleted, which means the gpu is freed.
	removeChan <-chan *PodResourceUpdate
	// Signal means the server is not healthy.
	nohealthChan     <-chan struct{}
	gpuinfoChan      <-chan *NodeGpuInfo
	nodeName         string
	prclient         podresourcesapi.PodResourcesListerClient
	grpconn          *grpc.ClientConn
	podresourcesLast map[string]*podresourcesapi.PodResources
	lastNodeGpuInfo  *NodeGpuInfo
	svcName          string
	gpuClient        gpuclientset.Interface
	gpuPodClient     gpupodcleintset.Interface
	gpuNodeLast      *gpunodev1.GpuNode
	gpuPodLast       map[string]*gpupodv1.GpuPod
	gpuPodLock       sync.RWMutex //used to protect gpuPodLast.
	once             sync.Once
}

func (dsc *ServerDSController) Start() error {
	relistChan := make(chan struct{}, 10)
	relistChan <- struct{}{}

	go func() {
		klog.Infof("ServerDSController started.")
	LOOP:
		for {
			select {
			case <-dsc.stop:
				break LOOP

			case ngi := <-dsc.gpuinfoChan:
				ngi.NodeName = dsc.nodeName
				dsc.lastNodeGpuInfo = ngi
				dsc.produceNodeGpuInfoCrd()
				atomic.SwapInt32(&serverdsutil.NodePushed, 1)

			case prupdate := <-dsc.removeChan:
				// del podresourcesLast
				if dsc.podresourcesLast != nil {
					podidx := strings.Join([]string{prupdate.PodResourcesDEL[0].Namespace, prupdate.PodResourcesDEL[0].Name}, "/")
					delete(dsc.podresourcesLast, podidx)
				}
				prupdate.NodeName = dsc.nodeName
				dsc.cleanPodResourceCrd(util.MetadataToName(prupdate.PodResourcesDEL[0].Namespace, prupdate.PodResourcesDEL[0].Name))
				dsc.produceNodeGpuInfoCrd()

			case <-dsc.goonChan:
				relistChan <- struct{}{}
				continue

			case <-relistChan:
				klog.Infof("go on list")
				ctx, ctxcancal := context.WithTimeout(context.Background(), options.DefaultPodResourcesTimeoutList)
				resp, err := dsc.prclient.List(ctx, &podresourcesapi.ListPodResourcesRequest{})
				if err != nil {
					ctxcancal()
					klog.Errorf("ListPodResourcesRequest err: %v", err)
					break LOOP
				}
				ctxcancal()

				//report any according to
				if dsc.podresourcesLast == nil {
					dsc.podresourcesLast = make(map[string]*podresourcesapi.PodResources)
				}

				prmapNew := make(map[string]*podresourcesapi.PodResources)
				changed := dsc.updatePodResourceFunc(resp.PodResources, dsc.podresourcesLast, prmapNew)
				dsc.podresourcesLast = prmapNew
				if changed {
					// ensure gpuNode.Spec.NodeDeviceInUse fresh.
					dsc.produceNodeGpuInfoCrd()
				}

				dsc.once.Do(dsc.cleanPodResourceCrdInit)
			}
		}
		klog.Infof("ServerDSController stopped.")
		if err := dsc.grpconn.Close(); err != nil {
			klog.Errorf("grpc conn close: %v", err)
		}
	}()
	return nil
}

func (dsc *ServerDSController) updatePodResourceFunc(prlist []*podresourcesapi.PodResources, prmapOld, prmapNew map[string]*podresourcesapi.PodResources) bool {
	prlistFiltered := fileterPodResource(prlist)

	changed := false
	for _, pr := range prlistFiltered {
		podidx := strings.Join([]string{pr.Namespace, pr.Name}, "/")
		prmapNew[podidx] = pr.PodResources //add or update
		if !reflect.DeepEqual(prmapOld[podidx], pr.PodResources) {
			go dsc.producePodResourceCrd(pr)
			changed = true
		}
	}

	for _, pr := range prmapOld {
		//exist podresource
		podidx := strings.Join([]string{pr.Namespace, pr.Name}, "/")
		if _, exist := prmapNew[podidx]; !exist {
			go dsc.cleanPodResourceCrd(util.MetadataToName(pr.Namespace, pr.Name))
			changed = true
		}
	}

	return changed
}

//fileterPodResource filter which we need
func fileterPodResource(prlist []*podresourcesapi.PodResources) []*PodResourcesDetail {
	prlistFiltered := make([]*PodResourcesDetail, 0, len(prlist))
	for _, pr := range prlist {
		prdCD := make([]*ContainerResourcesDetail, 0, len(pr.Containers))
		prd := &PodResourcesDetail{PodResources: pr, ContainerDevices: &prdCD}
		hasDevice := false
		for _, c := range pr.Containers {
			if len(c.Devices) != 0 {
				hasDevice = true
				//each container may need more than one devices
				prdCDcrd := &ContainerResourcesDetail{Name: c.Name, DeviceInfo: make([]*GpuInfo, 0, 1)}
				//get device details
				for _, d := range c.Devices {
					if d.ResourceName == options.NVIDIAGPUResourceName {
						for _, did := range d.DeviceIds {
							//get gpuinfo
							gpuinfo, err := updateGpuInfo(did)
							if err != nil {
								klog.Errorf("Error fileterPodResource.updateGpuInfo:%v", err)
							}
							prdCDcrd.DeviceInfo = append(prdCDcrd.DeviceInfo, gpuinfo)
						}
						break //only process nvidia.com/gpu
					}
				}
				prdCD = append(prdCD, prdCDcrd)
			}
		}
		if hasDevice {
			prlistFiltered = append(prlistFiltered, prd)
		}
	}
	return prlistFiltered
}

func (dsc *ServerDSController) cleanPodResourceCrdInit() {
	klog.Infof("Clean gpupods once when start.")
	lsOpt := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", options.GPUPOD_ANNOTATION_TAG_Node, dsc.nodeName), ResourceVersion: "0"}
	gpList, err := dsc.gpuPodClient.GpupodV1().GpuPods(metadata.MetadataNamespace()).List(context.TODO(), lsOpt)
	if err != nil {
		klog.Errorf("clean gpupod when start err:%v", err)
		return
	}

	for _, gp := range gpList.Items {
		podidx := strings.Join([]string{gp.Spec.Namespace, gp.Spec.Name}, "/")
		if _, ok := dsc.podresourcesLast[podidx]; ok {
			continue
		}
		klog.Infof("clean gpupod:%s", podidx)
		dsc.cleanPodResourceCrd(gp.Name)
	}
}

func (dsc *ServerDSController) cleanPodResourceCrd(nameGpuPod string) {
	err := retry.OnError(updateBackoff,
		func(err error) bool {
			if err != nil {
				return true
			}
			return false
		}, func() error {
			return dsc.cleanGpuPod(nameGpuPod)
		})

	if err == nil {
		klog.Infof("node:%s cleanPodResourceCrd:%s notice time:%v ", dsc.nodeName, nameGpuPod, time.Now().Format(time.RFC3339))
	} else {
		klog.Errorf("node:%s cleanPodResourceCrd err:%v", dsc.nodeName, err)
	}
}

func (dsc *ServerDSController) cleanGpuPod(nameGpuPod string) error {
	dsc.gpuPodLock.Lock()
	delete(dsc.gpuPodLast, nameGpuPod)
	dsc.gpuPodLock.Unlock()
	err := dsc.gpuPodClient.GpupodV1().GpuPods(metadata.MetadataNamespace()).Delete(context.TODO(), nameGpuPod, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("failed to cleanGpuPod:%s/%s err:%v", metadata.MetadataNamespace(), nameGpuPod, err)
	}
	return err
}

func (dsc *ServerDSController) producePodResourceCrd(prd *PodResourcesDetail) {
	err := retry.OnError(updateBackoff,
		func(err error) bool {
			if err != nil {
				return true
			}
			return false
		}, func() error {
			err := dsc.ensureGpuPod(prd)
			if err != nil {
				klog.Errorf("failed to ensureGpuPod:%s/%s err:%v", prd.Namespace, prd.Name, err)
			}
			return err
		})

	if err == nil {
		klog.Infof("node:%s producePodResourceCrd:%s/%s notice time:%v ", dsc.nodeName, prd.Namespace, prd.Name, time.Now().Format(time.RFC3339))
	} else {
		klog.Errorf("node:%s producePodResourceCrd err:%v", dsc.nodeName, err)
	}
}

func (dsc *ServerDSController) ensureGpuPod(prd *PodResourcesDetail) error {
	var gpuPodLast *gpupodv1.GpuPod
	gpuPodUuid := util.MetadataToName(prd.Namespace, prd.Name)
	dsc.gpuPodLock.RLock()
	gpuPodLast = dsc.gpuPodLast[gpuPodUuid]
	dsc.gpuPodLock.RUnlock()

	if gpuPodLast == nil {
		return dsc.getAndUpdateGpuPod(gpuPodUuid, prd)
	}
	// dsc.gpuPodLast != nil means we can update directly
	gpuPod := serverdsutil.ToGpuPod(dsc.nodeName, gpuPodLast, prd)
	gpuPod, err := dsc.gpuPodClient.GpupodV1().GpuPods(metadata.MetadataNamespace()).Update(context.TODO(), gpuPod, metav1.UpdateOptions{})
	if err != nil {
		// resource be deleted or other conflicts
		klog.Errorf("failed to update GpuPod:%s/%s, error: %v", gpuPod.Spec.Namespace, gpuPod.Spec.Name, err)
		return dsc.getAndUpdateGpuPod(gpuPodUuid, prd)
	}

	dsc.gpuPodLock.Lock()
	dsc.gpuPodLast[gpuPodUuid] = gpuPod
	dsc.gpuPodLock.Unlock()
	return nil
}

func (dsc *ServerDSController) getAndUpdateGpuPod(gpuPodUuid string, prd *PodResourcesDetail) error {
	//get from kube-apiserver cache
	gpuPod, err := dsc.gpuPodClient.GpupodV1().GpuPods(metadata.MetadataNamespace()).Get(context.TODO(), util.MetadataToName(prd.Namespace, prd.Name), metav1.GetOptions{ResourceVersion: "0"})
	if apierrors.IsNotFound(err) {
		gpuPod = serverdsutil.ToGpuPod(dsc.nodeName, nil, prd)
		gpuPod, err = dsc.gpuPodClient.GpupodV1().GpuPods(metadata.MetadataNamespace()).Create(context.TODO(), gpuPod, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		dsc.gpuPodLock.Lock()
		dsc.gpuPodLast[gpuPodUuid] = gpuPod
		dsc.gpuPodLock.Unlock()
		return nil
	} else if err != nil {
		return err
	}

	gpuPod = serverdsutil.ToGpuPod(dsc.nodeName, gpuPod, prd)
	gpuPod, err = dsc.gpuPodClient.GpupodV1().GpuPods(metadata.MetadataNamespace()).Update(context.TODO(), gpuPod, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	dsc.gpuPodLock.Lock()
	dsc.gpuPodLast[gpuPodUuid] = gpuPod
	dsc.gpuPodLock.Unlock()
	return nil
}

func (dsc *ServerDSController) produceNodeGpuInfoCrd() {
	err := retry.OnError(updateBackoff,
		func(err error) bool {
			if err != nil {
				return true
			}
			return false
		}, func() error {
			err := dsc.ensureGpuNode()
			if err != nil {
				klog.Errorf("failed to ensureGpuNode err:%v", err)
			}
			return err
		})

	if err == nil {
		klog.Infof("node:%s produceNodeGpuInfoCrd notice time:%v ", dsc.nodeName, time.Now().Format(time.RFC3339))
	} else {
		klog.Errorf("node:%s produceNodeGpuInfoCrd err:%v", dsc.nodeName, err)
	}
}

func (dsc *ServerDSController) ensureGpuNode() error {
	if dsc.gpuNodeLast == nil {
		return dsc.getAndUpdateGpuNode(dsc.lastNodeGpuInfo)
	}
	// dsc.gpuNodeLast != nil means we can update directly
	gpuNode := serverdsutil.ToGpuNode(dsc.nodeName, dsc.gpuNodeLast, dsc.lastNodeGpuInfo, dsc.podresourcesLast)
	gpuNode, err := dsc.gpuClient.GpunodeV1().GpuNodes(metadata.MetadataNamespace()).Update(context.TODO(), gpuNode, metav1.UpdateOptions{})
	if err != nil {
		// resource be deleted or other conflicts
		klog.Errorf("failed to update GpuNode:%s/%s, error: %v", gpuNode.Namespace, gpuNode.Name, err)
		return dsc.getAndUpdateGpuNode(dsc.lastNodeGpuInfo)
	}
	dsc.gpuNodeLast = gpuNode
	return nil
}

func (dsc *ServerDSController) getAndUpdateGpuNode(ngi *NodeGpuInfo) error {
	//get from kube-apiserver cache
	gpuNode, err := dsc.gpuClient.GpunodeV1().GpuNodes(metadata.MetadataNamespace()).Get(context.TODO(), dsc.nodeName, metav1.GetOptions{ResourceVersion: "0"})
	if apierrors.IsNotFound(err) {
		gpuNode = serverdsutil.ToGpuNode(dsc.nodeName, dsc.gpuNodeLast, ngi, dsc.podresourcesLast)
		gpuNode, err = dsc.gpuClient.GpunodeV1().GpuNodes(metadata.MetadataNamespace()).Create(context.TODO(), gpuNode, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		dsc.gpuNodeLast = gpuNode
		return nil
	} else if err != nil {
		return err
	}

	gpuNode = serverdsutil.ToGpuNode(dsc.nodeName, gpuNode, ngi, dsc.podresourcesLast)
	gpuNode, err = dsc.gpuClient.GpunodeV1().GpuNodes(metadata.MetadataNamespace()).Update(context.TODO(), gpuNode, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	dsc.gpuNodeLast = gpuNode
	return nil
}

// update ttlCacheGpu with node gpu info if device id is not exist
func updateGpuInfo(did string) (*GpuInfo, error) {
	gpuinfo := ttlCacheGpu.GetCacheGpuInfoIgnoreTTL(did)
	if gpuinfo != nil {
		return gpuinfo, nil
	}

	gpuinfo = &GpuInfo{DeviceId: did, NodeName: os.Getenv("NODENAME")}

	device, ret := nvml.DeviceGetHandleByUUID(did)
	if ret != nvml.SUCCESS {
		return gpuinfo, fmt.Errorf("DevicdId:%s DeviceGetHandleByUUID error: %v", did, nvml.ErrorString(ret))
	}
	//brand
	brandid, ret := device.GetBrand()
	if ret != nvml.SUCCESS {
		return gpuinfo, fmt.Errorf("device.GetBrand error: %v", nvml.ErrorString(ret))
	}
	gpuinfo.Brand = brand2type[brandid]
	//model
	gpuinfo.Model, ret = device.GetName()
	if ret != nvml.SUCCESS {
		return gpuinfo, fmt.Errorf("device.GetName error: %v", nvml.ErrorString(ret))
	}
	//pci busid
	pciinfo, ret := device.GetPciInfo()
	if ret != nvml.SUCCESS {
		return gpuinfo, fmt.Errorf("DeviceGetHandleByUUID error: %v", nvml.ErrorString(ret))
	}
	pciinfoBusid := make([]byte, 0, 32)
	for _, v := range pciinfo.BusId {
		if v != 0 {
			pciinfoBusid = append(pciinfoBusid, byte(v))
		}
	}
	gpuinfo.BusId = string(pciinfoBusid)

	ttlCacheGpu.SetCacheGpuInfo(did, &serverdsutil.CacheGpuInfo{GpuInfo: gpuinfo, LastUpdateTime: time.Now()})
	klog.V(4).Infof("DevicdId:%s, refresh gpu info from ttlCacheGpu:%#v GpuInfo:%#v", did, ttlCacheGpu, *(gpuinfo))
	return gpuinfo, nil
}
