package controllertype

import (
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
)

type WatcherInfo struct {
	ChanToAdd chan *PodResourceUpdate
	WatchUUID string
}
