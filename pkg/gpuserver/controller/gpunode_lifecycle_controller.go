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

// gpunode-lifecycle-controller will check the lease from each gpuserver-ds node to guarantee the health of gpunode,
// This controller should be started in leader mode, checking lease renew time to be fresh.
// if not, change the status of gpunode.

package controller

import (
	"context"
	"fmt"
	"time"

	gpunodev1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	gpuclientset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpunode/clientset/versioned"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	coordlisters "k8s.io/client-go/listers/coordination/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lifecycleControllerName  = LeaderResourceLockName
	LeaderResourceLockName   = "gpunode-lifecycle-controller"
	RetrySleepTime           = 20 * time.Millisecond
	GpuNodeHealthUpdateRetry = 3
)

func NewNodeLifecycleController(
	nodeMonitorPeriod time.Duration,
	nodeMonitorGracePeriod time.Duration,
	leaseLister coordlisters.LeaseLister,
	leaseInformerSynced cache.InformerSynced,
	gpuclient gpuclientset.Interface,
	gpuMgrClient client.Client,
	gpuInformerSynced cache.InformerSynced,
) *Controller {

	return &Controller{
		nodeMonitorPeriod:      nodeMonitorPeriod,
		nodeMonitorGracePeriod: nodeMonitorGracePeriod,
		leaseLister:            leaseLister,
		leaseInformerSynced:    leaseInformerSynced,
		gpuclient:              gpuclient,
		gpuMgrClient:           gpuMgrClient,
		gpuInformerSynced:      gpuInformerSynced,
		now:                    metav1.Now,
		savedHealthDataMap:     make(map[string]*nodeHealthData),
	}
}

type nodeHealthData struct {
	probeTimestamp metav1.Time
	lease          *coordinationv1.Lease
}

type Controller struct {
	nodeMonitorPeriod      time.Duration
	nodeMonitorGracePeriod time.Duration
	leaseLister            coordlisters.LeaseLister
	leaseInformerSynced    cache.InformerSynced
	gpuclient              gpuclientset.Interface
	gpuMgrClient           client.Client
	gpuInformerSynced      cache.InformerSynced
	now                    func() metav1.Time
	savedHealthDataMap     map[string]*nodeHealthData
}

func (nc *Controller) Run(ctx context.Context) {
	klog.Infof("Starting %s", lifecycleControllerName)
	defer klog.Infof("Shutting down %s", lifecycleControllerName)

	if !cache.WaitForNamedCacheSync(lifecycleControllerName, ctx.Done(), nc.leaseInformerSynced, nc.gpuInformerSynced) {
		panic("gpunode-lifecycle-controller unable to sync caches")
	}

	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		if err := nc.monitorGpuNodeHealth(ctx); err != nil {
			klog.Errorf("Error monitoring node health: %v", err)
		}
	}, nc.nodeMonitorPeriod)

	<-ctx.Done()
}

func (nc *Controller) monitorGpuNodeHealth(ctx context.Context) error {
	gpunodes := &gpunodev1.GpuNodeList{}
	err := nc.gpuMgrClient.List(ctx, gpunodes)
	if err != nil {
		return err
	}

	for _, gpunode := range gpunodes.Items {
		//list objects already deepcopyed.
		saveNodeHealth := nc.savedHealthDataMap[gpunode.Name]

		if err := wait.PollImmediate(RetrySleepTime, RetrySleepTime*GpuNodeHealthUpdateRetry, func() (bool, error) {
			if saveNodeHealth == nil {
				saveNodeHealth = &nodeHealthData{
					probeTimestamp: nc.now(),
				}
			}

			observedLease, err := nc.leaseLister.Leases(options.NamespaceNodeLease).Get(gpunode.Name)
			if err != nil {
				klog.Errorf("Get Leases of '%s', error: %v", gpunode.Name, err)
				return false, nil
			}

			if observedLease != nil && (saveNodeHealth.lease == nil || saveNodeHealth.lease.Spec.RenewTime.Before(observedLease.Spec.RenewTime)) {
				saveNodeHealth.lease = observedLease
				saveNodeHealth.probeTimestamp = nc.now()
			}
			nc.savedHealthDataMap[gpunode.Name] = saveNodeHealth

			//klog.Infof("saveNodeHealth.probeTimestamp:%v", saveNodeHealth.probeTimestamp)
			if nc.now().After(saveNodeHealth.probeTimestamp.Add(nc.nodeMonitorGracePeriod)) {
				// gpunode Leases not report health for nodeMonitorPeriod, so update gpunode status to false
				err = nc.updateGpuNodeStatus(ctx, false, &gpunode,
					fmt.Sprintf("Lease of GpuNode is not refreshed in %s.", nc.nodeMonitorGracePeriod.String()))
				if err != nil {
					klog.Errorf("Update GpuNodeStatusFail of '%s',error: %v", gpunode.Name, err)
					return false, nil
				}
			} else {
				err = nc.updateGpuNodeStatus(ctx, true, &gpunode,
					fmt.Sprintf("Lease of GpuNode is refreshed in %s.", nc.nodeMonitorGracePeriod.String()))
				if err != nil {
					klog.Errorf("Update GpuNodeStatusFail of '%s',error: %v", gpunode.Name, err)
					return false, nil
				}
			}

			return true, nil
		}); err != nil {
			klog.Errorf("Update status of GpuNode '%v' error: %v", gpunode.Name, err)
			continue
		}
	}

	return nil
}

func (nc *Controller) updateGpuNodeStatus(ctx context.Context, healthy bool, gpunode *gpunodev1.GpuNode, msg string) error {

	if healthy {
		if gpunode.Status.Health == gpunodev1.StatusHealth {
			// If already set health, skip set again.
			return nil
		}

		gpunode.Status.Message = msg
		gpunode.Status.Health = gpunodev1.StatusHealth
		gpunode.Status.LastTransitionTime = metav1.Now()
		gpunode.Status.LastHealthyTime = metav1.Now()

		upctx, cancelFun := context.WithTimeout(ctx, time.Second*2)
		defer cancelFun()
		_, err := nc.gpuclient.GpunodeV1().GpuNodes(gpunode.Namespace).UpdateStatus(upctx, gpunode, metav1.UpdateOptions{})
		return err
	}

	if gpunode.Status.Health == gpunodev1.StatusNotHealth {
		// If already set unhealth, skip set again.
		return nil
	}

	gpunode.Status.Message = msg
	gpunode.Status.Health = gpunodev1.StatusNotHealth
	gpunode.Status.LastTransitionTime = metav1.Now()
	if gpunode.Status.LastHealthyTime.IsZero() {
		gpunode.Status.LastHealthyTime = metav1.Now()
	}

	upctx, cancelFun := context.WithTimeout(ctx, time.Second*2)
	defer cancelFun()
	_, err := nc.gpuclient.GpunodeV1().GpuNodes(gpunode.Namespace).UpdateStatus(upctx, gpunode, metav1.UpdateOptions{})
	return err
}
