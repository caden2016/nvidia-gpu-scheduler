package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver-ds/podresources"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	serverdsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/serverds"
	"google.golang.org/grpc"
	"k8s.io/klog"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

var (
	brand2type = [18]string{"BRAND_UNKNOWN", "BRAND_QUADRO", "BRAND_TESLA", "BRAND_NVS", "BRAND_GRID", "BRAND_GEFORCE",
		"BRAND_TITAN", "BRAND_NVIDIA_VAPPS", "BRAND_NVIDIA_VPC", "BRAND_NVIDIA_VCS", "BRAND_NVIDIA_VWS", "BRAND_NVIDIA_VGAMING",
		"BRAND_QUADRO_RTX", "BRAND_NVIDIA_RTX", "BRAND_NVIDIA", "BRAND_GEFORCE_RTX", "BRAND_TITAN_RTX", "BRAND_COUNT"}
	ttlCacheGpu = serverdsutil.NewTTLCacheGpu(5 * time.Second)
)

func NewServerDSController(cacert []byte, goonChan <-chan struct{}, removeChan <-chan *PodResourceUpdate, nohealthChan <-chan struct{}, gpuinfoChan <-chan *NodeGpuInfo, healthChecker *HealthyChecker, podresourcesep string, stop <-chan struct{}) (*ServerDSController, error) {
	nodeName := os.Getenv("NODENAME")
	if nodeName == "" {
		return nil, fmt.Errorf("unable get env NODENAME")
	}

	client, conn, err := podresources.GetClient(podresourcesep, options.DefaultPodResourcesTimeoutConnect, options.DefaultPodResourcesMaxSize)
	if err != nil {
		return nil, fmt.Errorf("error getting grpc client: %v", err)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(cacert); !ok {
		return nil, err
	}

	dsc := &ServerDSController{
		goonChan:      goonChan,
		removeChan:    removeChan,
		stop:          stop,
		nohealthChan:  nohealthChan,
		gpuinfoChan:   gpuinfoChan,
		nodeName:      nodeName,
		healthChecker: healthChecker,
		prclient:      client,
		grpconn:       conn,
		discoveryClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: certPool,
				},
			},
			// the request should happen quickly.
			Timeout: 5 * time.Second,
		},
	}
	_, _, dsc.svcName = util.GetServiceCommonName()

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
	healthChecker    *HealthyChecker
	prclient         podresourcesapi.PodResourcesListerClient
	grpconn          *grpc.ClientConn
	discoveryClient  *http.Client
	podresourcesLast map[string]*podresourcesapi.PodResources
	lastNodeGpuInfo  *NodeGpuInfo
	svcName          string
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

			case <-dsc.nohealthChan:
				// means podresource service is unhealthy, maybe restart, before go to try service health again
				// we need to set podresourcesLast= nil, to make sure reproduce the latest podresources to the podresource service.
				klog.Error("HealthChecker signal not health, CheckHealthBlock again")
				dsc.healthChecker.CheckHealthBlock()
				klog.Error("HealthChecker signal health again, start CheckHealth")
				dsc.nohealthChan = dsc.healthChecker.CheckHealth(options.HealthChecker_CheckHealthInterval)
				dsc.podresourcesLast = nil
				relistChan <- struct{}{}
				// need to republish node gpu info after the gpuserver is healthy again.
				dsc.produceNodeGpuInfo(dsc.discoveryClient, dsc.lastNodeGpuInfo)

			case ngi := <-dsc.gpuinfoChan:
				ngi.NodeName = dsc.nodeName
				dsc.lastNodeGpuInfo = ngi
				dsc.produceNodeGpuInfo(dsc.discoveryClient, ngi)

			case prupdate := <-dsc.removeChan:
				// del podresourcesLast
				if dsc.podresourcesLast != nil {
					podidx := strings.Join([]string{prupdate.PodResourcesDEL[0].Namespace, prupdate.PodResourcesDEL[0].Name}, "/")
					delete(dsc.podresourcesLast, podidx)
				}
				prupdate.NodeName = dsc.nodeName
				dsc.producePodResource(dsc.discoveryClient, prupdate)

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
				prchanged, prupdate := updatePodResourceFunc(resp.PodResources, dsc.podresourcesLast, prmapNew)
				dsc.podresourcesLast = prmapNew
				prupdate.NodeName = dsc.nodeName

				if prchanged {
					dsc.producePodResource(dsc.discoveryClient, prupdate)
				}
			}
		}
		klog.Infof("ServerDSController stopped.")
		if err := dsc.grpconn.Close(); err != nil {
			klog.Errorf("grpc conn close: %v", err)
		}
	}()
	return nil
}

func updatePodResourceFunc(prlist []*podresourcesapi.PodResources, prmapOld, prmapNew map[string]*podresourcesapi.PodResources) (bool, *PodResourceUpdate) {
	prupdate := &PodResourceUpdate{PodResourcesSYNC: make([]*PodResourcesDetail, 0), PodResourcesDEL: make([]*podresourcesapi.PodResources, 0)}
	prlistFiltered := fileterPodResource(prlist)

	podresourcesListChanged := false

	for _, pr := range prlistFiltered {
		podidx := strings.Join([]string{pr.Namespace, pr.Name}, "/")
		prmapNew[podidx] = pr.PodResources //add or update
		prupdate.PodResourcesSYNC = append(prupdate.PodResourcesSYNC, pr)
		if !reflect.DeepEqual(prmapOld[podidx], pr.PodResources) {
			podresourcesListChanged = true
		}
	}

	for _, pr := range prmapOld {
		//exist podresource
		podidx := strings.Join([]string{pr.Namespace, pr.Name}, "/")
		if _, exist := prmapNew[podidx]; !exist {
			prupdate.PodResourcesDEL = append(prupdate.PodResourcesDEL, pr)
			podresourcesListChanged = true
		}
	}

	// Case:123001 len(prlistFiltered) == 0
	// Scenario all the gpu pods is deleted during the serverds is down.
	// When the serverds up again, it may miss the deleted gpu pod notification.
	// But when some of gpu pod is deleted during the serverds is down, will send PodResourcesSYNC with the current gpu pod.
	if len(prlistFiltered) == 0 {
		podresourcesListChanged = true
	}
	return podresourcesListChanged, prupdate
}

//fileterPodResource filter which we need
func fileterPodResource(prlist []*podresourcesapi.PodResources) []*PodResourcesDetail {
	prlistFiltered := make([]*PodResourcesDetail, 0, len(prlist))
	for _, pr := range prlist {
		prd_cd := make([]*ContainerResourcesDetail, 0, len(pr.Containers))
		prd := &PodResourcesDetail{PodResources: pr, ContainerDevices: &prd_cd}
		hasDevice := false
		for _, c := range pr.Containers {
			if len(c.Devices) != 0 {
				hasDevice = true
				//each container may need more than one devices
				prd_cd_crd := &ContainerResourcesDetail{Name: c.Name, DeviceInfo: make([]*GpuInfo, 0, 1)}
				//get device details
				for _, d := range c.Devices {
					if d.ResourceName == options.NVIDIAGPUResourceName {
						for _, did := range d.DeviceIds {
							//get gpuinfo
							gpuinfo, err := updateGpuInfo(did)
							if err != nil {
								klog.Errorf("Error fileterPodResource.updateGpuInfo:%v", err)
							}
							prd_cd_crd.DeviceInfo = append(prd_cd_crd.DeviceInfo, gpuinfo)
						}
						break //only process nvidia.com/gpu
					}
				}
				prd_cd = append(prd_cd, prd_cd_crd)
			}
		}
		if hasDevice {
			prlistFiltered = append(prlistFiltered, prd)
		}
	}
	return prlistFiltered
}

func (dsc *ServerDSController) producePodResource(discoveryClient *http.Client, data interface{}) {
	urlstr := fmt.Sprintf("https://%s/podresources", dsc.svcName)
	jsonndata, _ := json.Marshal(data)
	errcount := 1
	for errcount < options.PRODUCE_MAX_ERRORCOUNT {
		_, err := discoveryClient.Post(urlstr, "application/json", bytes.NewReader(jsonndata))
		if err != nil {
			klog.Errorf("ProducePodResource err: %v", err)
			errcount++
			time.Sleep(time.Duration(errcount) * time.Second)
		} else {
			break
		}
	}
	if errcount < options.PRODUCE_MAX_ERRORCOUNT {
		klog.Infof("ProducePodResource notice time:%v PodResources:%s", time.Now().Format(time.RFC3339), string(jsonndata))
	}
}

func (dsc *ServerDSController) produceNodeGpuInfo(discoveryClient *http.Client, data interface{}) {
	// Need not retry, since HostGpuInfoChecker produces each interval.
	urlstr := fmt.Sprintf("https://%s/hostgpuinfo", dsc.svcName)
	jsonndata, _ := json.Marshal(data)
	_, err := discoveryClient.Post(urlstr, "application/json", bytes.NewReader(jsonndata))
	if err != nil {
		klog.Errorf("ProduceNodeGpuInfo err: %v", err)
	} else {
		klog.Infof("ProduceNodeGpuInfo notice time:%v PodResources:%s", time.Now().Format(time.RFC3339), string(jsonndata))
	}
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
	pciinfo_busid := make([]byte, 0, 32)
	for _, v := range pciinfo.BusId {
		if v != 0 {
			pciinfo_busid = append(pciinfo_busid, byte(v))
		}
	}
	gpuinfo.BusId = string(pciinfo_busid)

	ttlCacheGpu.SetCacheGpuInfo(did, &serverdsutil.CacheGpuInfo{GpuInfo: gpuinfo, LastUpdateTime: time.Now()})
	klog.V(4).Infof("DevicdId:%s, refresh gpu info from ttlCacheGpu:%#v GpuInfo:%#v", did, ttlCacheGpu, *(gpuinfo))
	return gpuinfo, nil
}
