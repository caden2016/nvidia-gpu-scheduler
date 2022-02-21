package server

import (
	"context"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func GetPodRequestGpuNum(pod *corev1.Pod) int64 {
	var numLimit int64
	for _, c := range pod.Spec.Containers {
		if c.Resources.Limits != nil {
			if gpulimit, exist := c.Resources.Limits[options.NVIDIAGPUResourceName]; exist {
				if !gpulimit.IsZero() {
					numLimit += gpulimit.Value()
				}
			}
		}
	}
	return numLimit
}

func MapSetToList(mapset map[string]sets.String) map[string][]string {
	r := make(map[string][]string)
	for k, v := range mapset {
		r[k] = v.List()
	}
	return r
}

func GetGpuNodeHasSynced(ctx context.Context, mgr manager.Manager, object client.Object) (cache.InformerSynced, error) {
	informer, err := mgr.GetCache().GetInformer(ctx, object)
	if err != nil {
		return nil, err
	}
	return informer.HasSynced, nil
}
