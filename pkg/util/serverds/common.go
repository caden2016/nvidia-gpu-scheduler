package serverds

import (
	"fmt"
	"strings"
	"time"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"

	gpunodev1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	gpupodv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"

	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"

	"k8s.io/apimachinery/pkg/util/sets"
)

func DumpModelSetInfo(modelset map[string]sets.String) string {
	sb := strings.Builder{}
	for k, v := range modelset {
		sb.WriteString(fmt.Sprintf("[model:%s,sets:%v] ", k, v.List()))
	}
	return sb.String()
}

func ToGpuNode(nodeName string, base *gpunodev1.GpuNode, ngi *jsonstruct.NodeGpuInfo, prm map[string]*podresourcesapi.PodResources) *gpunodev1.GpuNode {
	var gpuNode *gpunodev1.GpuNode

	if base == nil {
		gpuNode = &gpunodev1.GpuNode{}
		gpuNode.Name = nodeName
	} else {
		gpuNode = base.DeepCopy()
	}

	if ngi != nil {
		gpuNode.Spec.GpuInfos = ngi.GpuInfos
		gpuNode.Spec.Models = mapSetToList(ngi.Models)
		gpuNode.Spec.ReportTime = metav1.Now()
		gpuNode.Status.NodeName = ngi.NodeName
	}

	if prm != nil {
		gpuNode.Spec.NodeDeviceInUse = getBusyDeviceSet(prm)
	}

	gpuNode.Status.LastHealthyTime = metav1.Now()
	gpuNode.Status.LastTransitionTime = metav1.Now()
	gpuNode.Status.Message = "Just Populate from gpuserver-ds."

	return gpuNode
}

func ToGpuPod(nodeName string, base *gpupodv1.GpuPod, prd *jsonstruct.PodResourcesDetail) *gpupodv1.GpuPod {
	var gpuPod *gpupodv1.GpuPod

	if base == nil {
		gpuPod = &gpupodv1.GpuPod{}
		gpuPod.Name = util.MetadataToName(prd.Namespace, prd.Name)
		gpuPod.Spec.Namespace = prd.Namespace
		gpuPod.Spec.Name = prd.Name
		gpuPod.Labels = make(map[string]string)
		gpuPod.Labels[options.GPUPOD_ANNOTATION_TAG_Node] = nodeName
		gpuPod.Spec.NodeName = nodeName
	} else {
		gpuPod = base.DeepCopy()
	}

	gpuPod.Spec.ContainerDevices = nil
	for _, crd := range *(prd.ContainerDevices) {
		gpuPod.Spec.ContainerDevices = append(gpuPod.Spec.ContainerDevices, gpupodv1.ContainerResourcesDetail(*crd))
	}
	gpuPod.Status.LastChangedTime = time.Now().Format(time.RFC3339)

	return gpuPod
}

func mapSetToList(mapset map[string]sets.String) map[string][]string {
	r := make(map[string][]string)
	for k, v := range mapset {
		r[k] = v.List()
	}
	return r
}

func getBusyDeviceSet(prm map[string]*podresourcesapi.PodResources) []string {
	deviceList := make([]string, 0)
	for _, pr := range prm {
		for _, cr := range pr.Containers {
			for _, device := range cr.Devices {
				for _, did := range device.DeviceIds {
					deviceList = append(deviceList, did)
				}
			}
		}
	}
	return deviceList
}
