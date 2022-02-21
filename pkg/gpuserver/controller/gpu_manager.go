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
	"fmt"
	"math/rand"
	"os"
	"time"

	"k8s.io/client-go/tools/leaderelection"

	gpunodev1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	gpupodv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	gpuclientset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpunode/clientset/versioned"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"

	"github.com/tal-tech/go-zero/core/sysx"
	v1 "k8s.io/api/core/v1"
	leaseinformer "k8s.io/client-go/informers/coordination/v1"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	leaseLister "k8s.io/client-go/listers/coordination/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

const (
	metricsAddr = ":8082"
	probeAddr   = ":8083"
	webhookPort = 9443
	// enableLeaderElection should be false in gpunode reconciler. All gpuserver will be notified.
	enableLeaderElection = false
	exitCode             = 100
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gpunodev1.AddToScheme(scheme))
	utilruntime.Must(gpupodv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func StartGpuManagerAndLifecycleControllerErrExit(ctx context.Context, kubeconfig *rest.Config, kubeClient kubernetes.Interface, gpuClient gpuclientset.Interface) (gpuMgrClient client.Client) {
	opts := zap.Options{
		Development: false,
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   webhookPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "37cd4f3d.caden2016.github.io",
	})
	if err != nil {
		klog.ErrorS(err, "unable to start manager")
		os.Exit(exitCode)
	}

	if err = (&GpuNodeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "unable to create controller", "controller", "GpuNode")
		os.Exit(exitCode)
	}
	if err = (&GpuPodReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		klog.ErrorS(err, "unable to create controller", "controller", "GpuPod")
		os.Exit(exitCode)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.ErrorS(err, "unable to set up health check")
		os.Exit(exitCode)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.ErrorS(err, "unable to set up ready check")
		os.Exit(exitCode)
	}

	go func() {
		if err := startNLCC(ctx, mgr, kubeconfig, kubeClient, gpuClient); err != nil {
			klog.Error(err)
			os.Exit(exitCode)
		}
	}()

	go func() {
		klog.Info("starting reconciler controllers")
		if err := mgr.Start(ctx); err != nil {
			klog.ErrorS(err, "problem running manager")
			os.Exit(exitCode)
		}
	}()

	gpuMgrClient = mgr.GetClient()
	return
}

func createRecorder(kubeClient kubernetes.Interface, userAgent string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	return eventBroadcaster.NewRecorder(scheme, v1.EventSource{Component: userAgent})
}

func ResyncPeriod() func() time.Duration {
	MinResyncPeriod := time.Hour * 12
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(MinResyncPeriod.Nanoseconds()) * factor)
	}
}

func startNLCC(ctx context.Context, mgr manager.Manager, kubeconfig *rest.Config, kubeClient kubernetes.Interface, gpuClient gpuclientset.Interface) error {

	gpuNodeHasSynced, err := serverutil.GetGpuNodeHasSynced(ctx, mgr, &gpunodev1.GpuNode{})
	if err != nil {
		return err
	}

	nodeMonitorPeriod := time.Millisecond * 100
	NodeName := sysx.Hostname()
	LeaseDuration := 15 * time.Second
	RenewDeadline := 10 * time.Second
	RetryPeriod := 2 * time.Second
	nodeMonitorGracePeriod := 4 * time.Second // 2* renewInterval in start_lease_controller.go

	leaseInformer := leaseinformer.NewFilteredLeaseInformer(kubeClient, options.NamespaceNodeLease, ResyncPeriod()(), cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, nil)
	go leaseInformer.Run(ctx.Done())

	nlcc := NewNodeLifecycleController(
		nodeMonitorPeriod,
		nodeMonitorGracePeriod,
		leaseLister.NewLeaseLister(leaseInformer.GetIndexer()),
		leaseInformer.HasSynced,
		gpuClient,
		mgr.GetClient(),
		gpuNodeHasSynced,
	)

	eventRecorder := createRecorder(kubeClient, LeaderResourceLockName)
	rl, err := resourcelock.NewFromKubeconfig(resourcelock.LeasesResourceLock,
		options.NamespaceNodeLease,
		LeaderResourceLockName,
		resourcelock.ResourceLockConfig{
			Identity:      NodeName,
			EventRecorder: eventRecorder,
		},
		kubeconfig,
		RenewDeadline)
	if err != nil {
		return fmt.Errorf("error creating lock: %w", err)
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: LeaseDuration,
		RenewDeadline: RenewDeadline,
		RetryPeriod:   RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: nlcc.Run,
			OnStoppedLeading: func() {
				klog.Infof("%s leaderelection lost", LeaderResourceLockName)
			},
		},
		WatchDog: nil,
		Name:     LeaderResourceLockName,
	})
	return err
}
