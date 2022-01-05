package controller

import (
	"context"
	"fmt"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"os"
	"time"
)

func NewPodWatcher(kubeclient kubernetes.Interface, stop <-chan struct{}) (*PodWatcher, error) {
	nodeName := os.Getenv("NODENAME")
	if nodeName == "" {
		return nil, fmt.Errorf("unable get env NODENAME")
	}

	return &PodWatcher{
		nodeName:   nodeName,
		kubeclient: kubeclient,
		stop:       stop,
		goonChan:   make(chan struct{}),
		removeChan: make(chan *PodResourceUpdate),
	}, nil
}

// PodWatcher watch the pod from of each node, filter by gpu and pod condition running.
type PodWatcher struct {
	nodeName   string
	kubeclient kubernetes.Interface
	stop       <-chan struct{}
	// Signal the server the gpu pod on the node is sync. which means the gpu info changed.
	goonChan chan struct{}
	// Signal the server the gpu pod on the node is deleted, which means the gpu is freed.
	removeChan chan *PodResourceUpdate
}

func (pw *PodWatcher) Start() error {
	wthcxt, wthcxtcancel := context.WithCancel(context.TODO())
	wthopt := metav1.ListOptions{FieldSelector: "spec.nodeName=" + pw.nodeName}
	podwatch, err := pw.kubeclient.CoreV1().Pods(metav1.NamespaceAll).Watch(wthcxt, wthopt)
	if err != nil {
		wthcxtcancel()
		return fmt.Errorf("unable Watch pods: %v", err)
	}

	go func() {
		defer wthcxtcancel()
		klog.Infof("PodWatcher started.")
		for {
			select {
			case rc, ok := <-podwatch.ResultChan():
				if !ok {
					klog.Errorf("podwatch stoped. sleep %v then reconnect", options.PodWatcher_WATCH_RECONNECT_INTERVAL)
					podwatch, err = pw.kubeclient.CoreV1().Pods(metav1.NamespaceAll).Watch(wthcxt, wthopt)
					if err != nil {
						klog.Errorf("unable Watch pods: %v", err)
					}
					time.Sleep(options.PodWatcher_WATCH_RECONNECT_INTERVAL)
					pw.goonChan <- struct{}{}
					continue
				}

				klog.V(1).Infof("[%v] resevent Type:%v,Object:%p", ok, rc.Type, rc.Object)
				switch rc.Type {
				case watch.Modified:
					pod, ispod := rc.Object.(*corev1.Pod)
					if ispod && isActivePod(pod) && isGpuPod(pod) {
						klog.Infof("got a new gpu pod running: %s/%s", pod.Namespace, pod.Name)
						pw.goonChan <- struct{}{}
						continue
					}
					if !ispod {
						klog.Errorf("not a pod")
					} else {
						klog.V(1).Infof("pod %s/%s status:%s", pod.Namespace, pod.Name, pod.Status.Phase)
					}

				case watch.Deleted:
					pod, ispod := rc.Object.(*corev1.Pod)
					if !ispod {
						klog.Errorf("watch Deleted, not a pod")
						continue
					}
					//care only about gpu pod
					if !isGpuPod(pod) {
						continue
					}
					klog.Infof("got a new gpu pod deleted: %s/%s", pod.Namespace, pod.Name)

					//notice remove
					prupdate := &PodResourceUpdate{PodResourcesSYNC: make([]*PodResourcesDetail, 0), PodResourcesDEL: []*podresourcesapi.PodResources{{}}}
					prupdate.PodResourcesDEL[0].Namespace = pod.Namespace
					prupdate.PodResourcesDEL[0].Name = pod.Name
					pw.removeChan <- prupdate
				}
			case <-pw.stop:
				klog.Info("PodWatcher exit with stop siganl")
			}
		}
	}()

	return nil
}

func (pw *PodWatcher) GetSyncChan() <-chan struct{} {
	return pw.goonChan
}

func (pw *PodWatcher) GetRemoveChan() <-chan *PodResourceUpdate {
	return pw.removeChan
}

func isActivePod(pod *corev1.Pod) bool {
	if pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
		return false
	}
	return true
}

func isGpuPod(pod *corev1.Pod) bool {
	isGpuPod := false

	for _, c := range pod.Spec.Containers {
		if c.Resources.Limits != nil {
			if gpulimit, exist := c.Resources.Limits[options.NVIDIAGPUResourceName]; exist {
				if !gpulimit.IsZero() {
					isGpuPod = true
					break
				}
			}
		}
		if c.Resources.Requests != nil {
			if gpureq, exist := c.Resources.Requests[options.NVIDIAGPUResourceName]; exist {
				if !gpureq.IsZero() {
					isGpuPod = true
					break
				}
			}
		}
	}
	return isGpuPod
}
