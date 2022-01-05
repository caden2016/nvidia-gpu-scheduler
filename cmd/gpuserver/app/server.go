package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	certsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/certs/util"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	routerinit "github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router/init"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"github.com/gorilla/mux"
	"github.com/openkruise/kruise/pkg/webhook/util/generator"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"net/http"
	"time"
)

func runserver(sflags *options.MetricsPodResourceFlags) (err error) {
	if len(sflags.WriteConfigTo) > 0 {
		return serverutil.LogOrWriteConfig(sflags.WriteConfigTo, sflags)
	}
	serverutil.DumpConfig(sflags)
	stop, cancelFunc := util.SetupSignalHandler()
	defer cancelFunc()

	//initialize certs - auto or use files
	var certs *generator.Artifacts
	var aggregatorClient clientset.Interface
	var kubeClient kubernetes.Interface
	if sflags.TLSAuto {
		klog.Infof("Generate certs for server automatically")
		kubeClient, aggregatorClient, err = serverutil.GetKubeAndAggregatorClientset()
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

	// create and start node
	nhc := controller.NewNodeHealthChecker(options.NodeHealthChecker_CheckInterval, options.NodeHealthChecker_NodeStatusTTL, stop)
	nhc.Start()

	// create and start Main channel controller.
	serverController, err := controller.NewServerController(nhc, sflags.Scheduler.Parallelism, stop)
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
