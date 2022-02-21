package watcher

import (
	"sync"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/signal"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var (
	GpuPodWatcher  Watcher
	GpuNodeWatcher Watcher
)

func init() {
	GpuPodWatcher = NewWatcher(GpuPodType)
	GpuNodeWatcher = NewWatcher(GpuNodeType)
	signal.AddCleanFuncs(GpuPodWatcher.CleanWatcher, GpuNodeWatcher.CleanWatcher)
}

type ChanType string

const (
	GpuNodeType ChanType = "GpuNode"
	GpuPodType  ChanType = "GpuPod"
)

func NewWatcher(chType ChanType) Watcher {
	return &WatchIndex{
		chanType:       chType,
		chanWatchIndex: make(map[string]chan interface{}),
	}
}

// WatchIndex provide AddWatcher, DelWatcher, ListWatcher to notify the change of gpupod and gpunode.
type WatchIndex struct {
	sync.RWMutex
	chanType       ChanType
	chanWatchIndex map[string]chan interface{}
}

func (w *WatchIndex) AddWatcher(chprd chan interface{}) string {
	watchUuid := string(uuid.NewUUID())
	w.Lock()
	defer w.Unlock()
	w.chanWatchIndex[watchUuid] = chprd
	return watchUuid
}

func (w *WatchIndex) DelWatcher(watchUuid string) {
	w.Lock()
	defer w.Unlock()
	delete(w.chanWatchIndex, watchUuid)
}

func (w *WatchIndex) ListWatcher() []chan interface{} {
	w.RLock()
	defer w.RUnlock()
	chList := make([]chan interface{}, 0, len(w.chanWatchIndex))
	for _, ch := range w.chanWatchIndex {
		chList = append(chList, ch)
	}
	return chList
}

// CleanWatcher close all watcher chan and empty chanWatchIndex, called when gpuserver exit.
func (w *WatchIndex) CleanWatcher() {
	w.Lock()
	defer w.Unlock()
	for _, ch := range w.chanWatchIndex {
		close(ch)
	}
	w.chanWatchIndex = make(map[string]chan interface{})
}
