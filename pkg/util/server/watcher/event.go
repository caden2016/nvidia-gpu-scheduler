package watcher

import (
	gpunodev1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	gpupodv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"
)

type EventType string

const (
	Deleted EventType = "DELETED"
	Synced  EventType = "SYNCED" //Added or Modified
)

type GpuNodeEvent struct {
	GpuNode *gpunodev1.GpuNode
	Type    EventType
}

func NewGpuNodeEvent(gpuNode *gpunodev1.GpuNode, etype EventType) *GpuNodeEvent {
	return &GpuNodeEvent{
		GpuNode: gpuNode,
		Type:    etype,
	}
}

type GpuPodEvent struct {
	GpuPod *gpupodv1.GpuPod
	Type   EventType
}

func NewGpuPodEvent(gpuPod *gpupodv1.GpuPod, etype EventType) *GpuPodEvent {
	return &GpuPodEvent{
		GpuPod: gpuPod,
		Type:   etype,
	}
}
