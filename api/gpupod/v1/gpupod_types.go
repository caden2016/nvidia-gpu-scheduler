/*
Copyright © 2021 The nvidia-gpu-scheduler Authors.
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GpuPodSpec defines the desired state of GpuPod
type GpuPodSpec struct {
	Name      string `json:"pod_name,omitempty"`
	Namespace string `json:"pod_namespace,omitempty"`
	NodeName  string `json:"node_name,omitempty"`
	// ContainerDevices is list of container name and gpu devices in each container.
	ContainerDevices []ContainerResourcesDetail `json:"containers_device,omitempty"`
}

type ContainerResourcesDetail struct {
	Name       string                `json:"container_name,omitempty"`
	DeviceInfo []*jsonstruct.GpuInfo `json:"device_info,omitempty"`
}

// GpuPodStatus defines the observed state of GpuPod
type GpuPodStatus struct {
	LastChangedTime string `json:"last_changed_time,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAMESPACE",type="string",JSONPath=".spec.pod_namespace",description="The pod namespace."
// +kubebuilder:printcolumn:name="PODNAME",type="string",JSONPath=".spec.pod_name",description="The pod name."
// +kubebuilder:printcolumn:name="NODE",type="string",JSONPath=".spec.node_name",description="The node name."
// +kubebuilder:printcolumn:name="UPDATE",type="string",JSONPath=".status.last_changed_time",description="The update time."

// GpuPod is the Schema for the gpupods API
type GpuPod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GpuPodSpec   `json:"spec,omitempty"`
	Status GpuPodStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GpuPodList contains a list of GpuPod
type GpuPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GpuPod `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GpuPod{}, &GpuPodList{})
}
