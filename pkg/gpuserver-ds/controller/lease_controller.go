package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	gpuclientset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpunode/clientset/versioned"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/info/metadata"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//+kubebuilder:scaffold:imports
	serverdsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/serverds"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/component-helpers/apimachinery/lease"
)

const (
	leaseControllerName = "node-lease-controller"
	exitCode            = 101
)

func StartLeaseControllerErrExit(ctx context.Context, kubeClient kubernetes.Interface, gpuClient gpuclientset.Interface) {
	if err := startLeaseController(ctx, kubeClient, gpuClient); err != nil {
		klog.Errorf("%s: %v", leaseControllerName, err)
		os.Exit(exitCode)
	}
}

func startLeaseController(ctx context.Context, kubeClient kubernetes.Interface, gpuClient gpuclientset.Interface) error {
	node := os.Getenv("NODENAME")
	if node == "" {
		return fmt.Errorf("unable get env NODENAME")
	}

	if err := ensureNamespace(ctx, kubeClient, options.NamespaceNodeLease); err != nil {
		return err
	}

	//NodeHealthChecker_NodeStatusTTL     := 3 * time.Second
	nodeLeaseRenewIntervalFraction := 0.25
	var NodeLeaseDurationSeconds int32 = 8
	leaseDuration := time.Duration(NodeLeaseDurationSeconds) * time.Second
	renewInterval := time.Duration(float64(leaseDuration) * nodeLeaseRenewIntervalFraction)
	nodeLeaseController := lease.NewController(
		clock.RealClock{},
		kubeClient,
		node,
		NodeLeaseDurationSeconds,
		nil,
		renewInterval,
		options.NamespaceNodeLease,
		serverdsutil.SetNodeOwnerFunc(gpuClient, metadata.MetadataNamespace(), node))

	klog.Infof("starting %s", leaseControllerName)
	go nodeLeaseController.Run(ctx.Done())
	return nil
}

func ensureNamespace(ctx context.Context, kubeclient kubernetes.Interface, nsname string) error {
	nsctx, cancelFun := context.WithTimeout(ctx, time.Second*2)
	defer cancelFun()
	_, err := kubeclient.CoreV1().Namespaces().Get(nsctx, nsname, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = kubeclient.CoreV1().Namespaces().Create(nsctx,
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsname}},
			metav1.CreateOptions{})
	}
	return err
}
