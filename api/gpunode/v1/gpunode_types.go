/*
Copyright 2022.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GpuNodeSpec defines the desired state of GpuNode
type GpuNodeSpec struct {
	// GpuInfos defines the observed state of gpu from each node.
	GpuInfos map[string]*jsonstruct.GpuInfo `json:"device_infos,omitempty"`
	// Models group the gpus by model.
	Models map[string][]string `json:"device_models,omitempty"`
	// NodeDeviceInUse defines the gpus which are used.
	NodeDeviceInUse []string `json:"device_busy"`
	// ReportTime record the time gpuinfo populated by each gpuserver-ds.
	ReportTime metav1.Time `json:"report_time,omitempty"`
}

// GpuNodeStatus defines the observed state of GpuNode.
// This will be updated with resource GpuNodeHealth.
type GpuNodeStatus struct {
	NodeName           string      `json:"node,omitempty"`
	Health             string      `json:"health,omitempty"`
	Message            string      `json:"message,omitempty"`
	LastHealthyTime    metav1.Time `json:"last_health_time,omitempty"`
	LastTransitionTime metav1.Time `json:"last_transition_time,omitempty"`
}

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="HEATH",type="string",JSONPath=".status.health",description="The status of node."
// +kubebuilder:printcolumn:name="LastHealthyTime",type="string",JSONPath=".status.last_health_time",description="The last healthy time of node."
// +kubebuilder:printcolumn:name="LastTransitionTime",type="string",JSONPath=".status.last_transition_time",description="The last transition time of node status."
// +kubebuilder:printcolumn:name="MESSAGE",type="string",JSONPath=".status.message",description="The status message of node."
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. Clients may not set this value. It is represented in RFC3339 form and is in UTC."

// GpuNode is the Schema for the gpunodes API
type GpuNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GpuNodeSpec   `json:"spec,omitempty"`
	Status GpuNodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GpuNodeList contains a list of GpuNode
type GpuNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GpuNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GpuNode{}, &GpuNodeList{})
}

const (
	StatusHealth    = "True"
	StatusNotHealth = "False"
)
