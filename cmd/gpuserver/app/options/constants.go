package options

const (
	CERTDIR               = `/tmp/ssl`
	CERT_Secret_Namespace = ServiceNamespace
	CERT_Secret_Name      = ServiceName

	ServiceNamespace = `kube-system`
	ServiceName      = `nvidia-gpu-scheduler`
	APIGROUP         = `nvidia-gpu-scheduler`
	APIVERSION       = `v1`
	RESOURCES_GPUPOD = `gpupods`
	RESOURCE_GPUPOD  = `gpupod`
	KIND_GPUPOD      = `GpuPod`

	SCHEDULE                            = `schedule`
	SCHEDULE_FILTER                     = `filter`
	SCHEDULE_PREEMPT                    = `preempt`
	SCHEDULE_PRIORITIZE                 = `prioritize`
	SCHEDULE_ANNOTATION                 = `nvidia-gpu-scheduler/gpu.model`
	RESOURCES_GPUNODE                   = `gpunodes`
	RESOURCE_GPUNODE                    = `gpunode`
	KIND_GPUNODE                        = `GpuNode`
	SchedulerRouter_Parallelism_Default = 10

	//v0.2.0
	NamespaceNodeLease = "nvidia-gpu-scheduler-node-lease"
)
