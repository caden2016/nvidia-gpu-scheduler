package options

import "time"

const (
	CERTDIR               = `/tmp/ssl`
	CERT_Secret_Namespace = ServiceNamespace
	CERT_Secret_Name      = ServiceName

	ServiceNamespace                    = `kube-system`
	ServiceName                         = `nvidia-gpu-scheduler`
	APIGROUP                            = `metrics.nvidia.com`
	APIVERSION                          = `v1alpha1`
	RESOURCES                           = `podresources`
	RESOURCE                            = `podresource`
	KIND                                = `PodResource`
	SCHEDULE                            = `schedule`
	SCHEDULE_FILTER                     = `filter`
	SCHEDULE_PREEMPT                    = `preempt`
	SCHEDULE_PRIORITIZE                 = `prioritize`
	SCHEDULE_ANNOTATION                 = `nvidia.com/gpu.model`
	GPURESOURCES                        = `gpuinfos`
	GPURESOURCE                         = `gpuinfo`
	GPUKIND                             = `GpuInfo`
	NodeHealthChecker_NodeStatusTTL     = 3 * time.Second // a little bigger than HealthChecker_CheckHealthInterval 2s
	NodeHealthChecker_CheckInterval     = 100 * time.Millisecond
	SchedulerRouter_Parallelism_Default = 10
)
