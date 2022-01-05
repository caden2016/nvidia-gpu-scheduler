package app

import (
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver-ds/controller"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	serverdsutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/serverds"
)

func runserverds(sflags *options.MetricsPodResourceDSFlags) (err error) {
	if len(sflags.WriteConfigTo) > 0 {
		return serverutil.LogOrWriteConfig(sflags.WriteConfigTo, sflags)
	}
	serverutil.DumpConfig(sflags)
	stop, cancelFunc := util.SetupSignalHandler()
	defer cancelFunc()

	hc := controller.NewHealthyChecker(sflags.IntervalWaitService, stop)

	hc.CheckHealthBlock()

	gic, err := controller.NewHostGpuInfoChecker(options.HostGpuInfoChecker_CheckInterval, stop)
	if err != nil {
		return err
	}

	kubeClient, _, err := serverutil.GetKubeAndAggregatorClientset()
	if err != nil {
		return err
	}

	// Try to get tls ca after gpuserver is health, this will be blocked until get cacert.
	cacert := serverdsutil.EnsureGetCaFromSecrets(kubeClient)

	pw, err := controller.NewPodWatcher(kubeClient, stop)
	if err != nil {
		return err
	}

	//start PodWatcher controller
	err = pw.Start()
	if err != nil {
		return err
	}

	//start HealthChecker controller every second
	notheathChan := hc.CheckHealth(options.HealthChecker_CheckHealthInterval)

	//start HostGpuInfoChecker controller
	err = gic.Start()
	if err != nil {
		return err
	}

	dsc, err := controller.NewServerDSController(cacert, pw.GetSyncChan(), pw.GetRemoveChan(), notheathChan,
		gic.GetGpuInfoChan(), hc, sflags.LocalPodResourcesEndpoint, stop)
	if err != nil {
		return err
	}

	//start ServerDSController controller
	dsc.Start()
	if err != nil {
		return err
	}

	<-stop
	return nil
}
