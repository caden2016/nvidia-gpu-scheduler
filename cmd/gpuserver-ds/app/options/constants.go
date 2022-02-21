package options

import "time"

const (
	NVIDIAGPUResourceName = "nvidia.com/gpu"

	// DefaultPodResourcesEndpoint is the path to the local endpoint serving the podresources GRPC service.
	DefaultPodResourcesEndpoint       = "unix:///var/lib/kubelet/pod-resources/kubelet.sock"
	DefaultPodResourcesTimeoutConnect = 10 * time.Second
	DefaultPodResourcesMaxSize        = 1024 * 1024 * 16 // 16 Mb
	DefaultPodResourcesTimeoutList    = 5 * time.Second
	PRODUCE_MAX_ERRORCOUNT            = 4

	PodWatcher_WATCH_RECONNECT_INTERVAL = time.Second

	HostGpuInfoChecker_CheckInterval = 2 * time.Second

	CaFromSecret_CheckInterval = time.Second

	GPUPOD_ANNOTATION_TAG_Node = "nvidia-gpu-scheduler.node"
)
