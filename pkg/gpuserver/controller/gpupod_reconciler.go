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

package controller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server/watcher"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	resourcesschedulerv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"
)

const podReconcilerControllerName = "gpupod-reconciler-controller"

// GpuPodReconciler reconciles a GpuPod object
type GpuPodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=resources.scheduler.caden2016.github.io,resources=gpupods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=resources.scheduler.caden2016.github.io,resources=gpupods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=resources.scheduler.caden2016.github.io,resources=gpupods/finalizers,verbs=update

func (r *GpuPodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	gpuPod, err := r.getGpuPod(ctx, req)
	if err != nil {
		// Error reading the object - requeue the request.
		klog.Errorf("Failed to get %s, err:%v", req.String(), err)
		return ctrl.Result{}, err
	}

	var etype watcher.EventType
	if gpuPod == nil {
		klog.Infof("%s be deleted.", req.String())
		gpuPod = &resourcesschedulerv1.GpuPod{ObjectMeta: metav1.ObjectMeta{Namespace: req.Namespace, Name: req.Name}}
		etype = watcher.Deleted
	} else {
		klog.Infof("%s be synced.", req.String())
		etype = watcher.Synced

	}

	notifyWatchersGpuPod(gpuPod, etype)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GpuPodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcesschedulerv1.GpuPod{}).
		Complete(r)
}

func (r *GpuPodReconciler) getGpuPod(ctx context.Context, req ctrl.Request) (gpuPod *resourcesschedulerv1.GpuPod, err error) {
	ctxtodo, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	gpuPod = &resourcesschedulerv1.GpuPod{}
	err = r.Get(ctxtodo, req.NamespacedName, gpuPod)
	if err != nil {
		if errors.IsNotFound(err) {
			// resource be deleted
			gpuPod = nil
			err = nil
			return
		}
		gpuPod = nil
		return
	}
	return
}

// notifyWatchersGpuPod notify the change of gpunode to all watchers from rest api in metricserver.
func notifyWatchersGpuPod(gpuPod *resourcesschedulerv1.GpuPod, etype watcher.EventType) {
	egn := watcher.NewGpuPodEvent(gpuPod, etype)
	for _, wch := range watcher.GpuPodWatcher.ListWatcher() {
		select {
		case wch <- egn:
		default:
		}
	}
}
