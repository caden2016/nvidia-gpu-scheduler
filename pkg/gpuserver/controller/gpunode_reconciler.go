/*
Copyright © 2021 The nvidia-gpu-scheduler Authors.
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// gpunode-reconciler-controller monitor gpunode and update in internal cache to be used.
// It notifies all watcher when the gpunode info changed.

package controller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server/cache"

	resourcesschedulerv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server/watcher"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const nodeReconcilerControllerName = "gpunode-reconciler-controller"

// GpuNodeReconciler reconciles a GpuNode object
type GpuNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=resources.scheduler.caden2016.github.io,resources=gpunodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=resources.scheduler.caden2016.github.io,resources=gpunodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=resources.scheduler.caden2016.github.io,resources=gpunodes/finalizers,verbs=update

func (r *GpuNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	gpuNode, err := r.getGpuNode(ctx, req)
	if err != nil {
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get %s, err:%v", req.String(), err)
		return ctrl.Result{}, err
	}

	var etype watcher.EventType
	if gpuNode == nil {
		klog.Infof("%s be deleted.", req.String())
		gpuNode = &resourcesschedulerv1.GpuNode{ObjectMeta: metav1.ObjectMeta{Namespace: req.Namespace, Name: req.Name}}
		etype = watcher.Deleted
	} else {
		klog.Infof("%s be synced.", req.String())
		etype = watcher.Synced

		// set GpuNode in cache for scheduler.
		cache.DefaultGpuNodeCache.SetGpuNode(gpuNode.Name, gpuNode)

	}

	notifyWatchersGpuNode(gpuNode, etype)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GpuNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcesschedulerv1.GpuNode{}).
		Complete(r)
}

func (r *GpuNodeReconciler) getGpuNode(ctx context.Context, req ctrl.Request) (gpuNode *resourcesschedulerv1.GpuNode, err error) {
	ctxtodo, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	gpuNode = &resourcesschedulerv1.GpuNode{}
	err = r.Get(ctxtodo, req.NamespacedName, gpuNode)
	if err != nil {
		if errors.IsNotFound(err) {
			// resource be deleted
			gpuNode = nil
			err = nil
			return
		}
		gpuNode = nil
		return
	}
	return
}

// notifyWatchersGpuNode notify the change of gpunode to all watchers from rest api in metricserver.
func notifyWatchersGpuNode(gpuNode *resourcesschedulerv1.GpuNode, etype watcher.EventType) {
	egn := watcher.NewGpuNodeEvent(gpuNode, etype)
	for _, wch := range watcher.GpuNodeWatcher.ListWatcher() {
		select {
		case wch <- egn:
		default:
		}
	}
}
