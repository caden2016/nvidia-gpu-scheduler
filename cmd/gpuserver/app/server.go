package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/signal"

	"k8s.io/client-go/rest"

	gpuclientset "github.com/caden2016/nvidia-gpu-scheduler/pkg/generated/gpunode/clientset/versioned"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	certsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/certs/util"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	routerinit "github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router/init"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"github.com/gorilla/mux"
	"github.com/openkruise/kruise/pkg/webhook/util/generator"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

func runserver(sflags *options.MetricsPodResourceFlags) (err error) {
	if len(sflags.WriteConfigTo) > 0 {
		return serverutil.LogOrWriteConfig(sflags.WriteConfigTo, sflags)
	}
	serverutil.DumpConfig(sflags)
	stopCtx, cancelFunc := signal.SetupSignalHandler()
	defer cancelFunc()
	stop := stopCtx.Done()

	//initialize certs - auto or use files
	var certs *generator.Artifacts
	var aggregatorClient clientset.Interface
	var kubeClient kubernetes.Interface
	var gpuClient gpuclientset.Interface
	var kubeconf *rest.Config
	if sflags.TLSAuto {
		klog.Infof("Generate certs for server automatically")
		kubeconf, kubeClient, aggregatorClient, gpuClient, _, err = serverutil.GetKubeAndAggregatorClientset()
		if err != nil {
			return err
		}
		certs, err = certsutil.InitializeCert(kubeClient)
		if err != nil {
			return err
		}

	} else {
		klog.Infof("Generate certs for server with TLSConfig:%#v", sflags.TLSConfig)
		certs, err = certsutil.InitializeCertWithFile(&sflags.TLSConfig)
	}
	if err != nil {
		return err
	}
	err = serverutil.EnsureAPIService(aggregatorClient, certs.CACert)
	if err != nil {
		return err
	}

	//get tls.Config from certs
	tlscfg, err := serverutil.GetTlsConfig(certs)
	if err != nil {
		return err
	}

	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", sflags.BindAddress, sflags.BindPort), tlscfg)
	if err != nil {
		return err
	}

	servermux := mux.NewRouter()
	srv := &http.Server{
		Handler: servermux,
	}

	// start gpunode reconctler controller ande gpunode lifecycle controller.
	gpuMgrClient := controller.StartGpuManagerAndLifecycleControllerErrExit(stopCtx, kubeconf, kubeClient, gpuClient)

	// create and start Main channel controller.
	serverController, err := controller.NewServerController(stop, sflags.Scheduler.Parallelism, gpuMgrClient)
	if err != nil {
		return err
	}

	// register all routes supports by the server
	routerinit.RegisterRoutes(servermux, serverController, sflags.EnableScheduler)

	idleConnsClosed := make(chan struct{})
	go func() {
		<-stop
		klog.Info("shutting down webhook server in 30s")
		tocontext, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		if err := srv.Shutdown(tocontext); err != nil {
			klog.Error(err, "error shutting down the HTTP server")
		}
		close(idleConnsClosed)
	}()

	klog.Infof("server started with listening addr:%s", listener.Addr().String())
	err = srv.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	klog.Info("server end")
	return nil
}
