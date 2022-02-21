package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
)

var cleanFuns []func()

func SetupSignalHandler() (stopCtx context.Context, cancelFunc context.CancelFunc) {
	stopCtx, cancelFunc = context.WithCancel(context.Background())
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sigRec := <-c
		klog.Infof("Server receive signal: %v", sigRec)
		for _, cleanFun := range cleanFuns {
			cleanFun()
		}
		cancelFunc()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()
	return
}

func AddCleanFuncs(cfs ...func()) {
	cleanFuns = append(cleanFuns, cfs...)
}
