package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func SetupSignalHandler() (stopCh <-chan struct{}, cancelFunc context.CancelFunc) {
	stop, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()
	return stop.Done(), cancel
}
