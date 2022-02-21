package app

import (
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver-ds/controller"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/signal"
)

func runserverds(sflags *options.MetricsPodResourceDSFlags) (err error) {
	if len(sflags.WriteConfigTo) > 0 {
		return serverutil.LogOrWriteConfig(sflags.WriteConfigTo, sflags)
	}
	serverutil.DumpConfig(sflags)
	stopCtx, cancelFunc := signal.SetupSignalHandler()
	defer cancelFunc()
	stop := stopCtx.Done()

	gic, err := controller.NewHostGpuInfoChecker(options.HostGpuInfoChecker_CheckInterval, stop)
	if err != nil {
		return err
	}

	_, kubeClient, _, gpuClient, gpuPodClient, err := serverutil.GetKubeAndAggregatorClientset()
	if err != nil {
		return err
	}

	pw, err := controller.NewPodWatcher(kubeClient, stop)
	if err != nil {
		return err
	}

	//start PodWatcher controller
	err = pw.Start()
	if err != nil {
		return err
	}

	controller.StartLeaseControllerErrExit(stopCtx, kubeClient, gpuClient)

	//start HostGpuInfoChecker controller
	err = gic.Start()
	if err != nil {
		return err
	}

	dsc, err := controller.NewServerDSController(stop, pw.GetSyncChan(), pw.GetRemoveChan(),
		gic.GetGpuInfoChan(), sflags.LocalPodResourcesEndpoint, gpuClient, gpuPodClient)
	if err != nil {
		return err
	}

	//start ServerDSController controller
	err = dsc.Start()
	if err != nil {
		return err
	}

	<-stop
	return nil
}
