// Package watcher notify the change of gpupod and gpunode to watchers from rest api in metricserver.
package watcher

type Watcher interface {
	AddWatcher(chResource chan interface{}) string
	DelWatcher(watchUuid string)
	ListWatcher() []chan interface{}
	CleanWatcher()
}
