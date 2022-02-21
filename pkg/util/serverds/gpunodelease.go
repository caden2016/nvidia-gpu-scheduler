package serverds

import (
	"context"
	"sync/atomic"

	gpuv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	gpuclientset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpunode/clientset/versioned"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

//NodePushed used to notice lease_controller when SetNodeOwnerFunc.
var NodePushed int32

// SetNodeOwnerFunc helps construct a newLeasePostProcessFunc which sets
// a node OwnerReference to the given lease object
func SetNodeOwnerFunc(c gpuclientset.Interface, nodeNS, nodeName string) func(lease *coordinationv1.Lease) error {
	return func(lease *coordinationv1.Lease) error {
		// Setting owner reference needs node's UID. Note that it is different from
		// kubelet.nodeRef.UID. When lease is initially created, it is possible that
		// the connection between master and node is not ready yet. So try to set
		// owner reference every time when renewing the lease, until successful.
		np := atomic.LoadInt32(&NodePushed)
		if len(lease.OwnerReferences) == 0 && np == 1 {
			if node, err := c.GpunodeV1().GpuNodes(nodeNS).Get(context.TODO(), nodeName, metav1.GetOptions{}); err == nil {
				lease.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: gpuv1.SchemeGroupVersion.WithKind("GpuNode").Version,
						Kind:       gpuv1.SchemeGroupVersion.WithKind("GpuNode").Kind,
						Name:       nodeName,
						UID:        node.UID,
					},
				}
			} else {
				klog.ErrorS(err, "Failed to get node when trying to set owner ref to the node lease", "node", klog.KRef("", nodeName))
				return err
			}
		}
		return nil
	}
}
