package serverds

import (
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"k8s.io/klog"
	"sync"
	"time"
)

func NewTTLCacheGpu(ttl time.Duration) *TTLCacheGpu {
	return &TTLCacheGpu{
		gpuinfo:  make(map[string]*CacheGpuInfo),
		cacheTTL: ttl,
		rwlock:   &sync.RWMutex{},
	}
}

// TTLCacheGpu store for GpuInfo, in case of calling getGpuInfo echo interval PRODUCE_INTERVAL
type TTLCacheGpu struct {
	gpuinfo  map[string]*CacheGpuInfo
	cacheTTL time.Duration
	rwlock   *sync.RWMutex
}

func (cg *TTLCacheGpu) GetCacheGpuInfoIgnoreTTL(did string) *GpuInfo {
	cg.rwlock.RLock()
	defer cg.rwlock.RUnlock()
	if cg.gpuinfo[did] != nil {
		klog.V(9).Infof("DevicdId:%s, get gpu info from ttlCacheGpu:%#v", did, *cg.gpuinfo[did])
		return cg.gpuinfo[did].GpuInfo
	}
	return nil
}

func (cg *TTLCacheGpu) GetCacheGpuInfo(did string) *GpuInfo {
	cg.rwlock.RLock()
	defer cg.rwlock.RUnlock()
	if cg.gpuinfo[did] != nil && cg.gpuinfo[did].isObjectFresh(cg.cacheTTL) {
		klog.V(9).Infof("DevicdId:%s, get gpu info from ttlCacheGpu:%#v", did, *cg.gpuinfo[did])
		return cg.gpuinfo[did].GpuInfo
	}
	return nil
}

func (cg *TTLCacheGpu) SetCacheGpuInfo(did string, cgpuinfo *CacheGpuInfo) {
	cg.rwlock.Lock()
	defer cg.rwlock.Unlock()
	cg.gpuinfo[did] = cgpuinfo
}

type CacheGpuInfo struct {
	*GpuInfo
	LastUpdateTime time.Time
}

func (cgi *CacheGpuInfo) isObjectFresh(cachettl time.Duration) bool {
	return time.Now().Before(cgi.LastUpdateTime.Add(cachettl))
}
