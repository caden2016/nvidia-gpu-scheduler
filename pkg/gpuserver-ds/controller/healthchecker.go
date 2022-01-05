package controller

import (
	"crypto/tls"
	"fmt"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util"
	"k8s.io/klog"
	"net/http"
	"os"
	"time"
)

func NewHealthyChecker(intervalWaitService time.Duration, stop <-chan struct{}) *HealthyChecker {
	hc := &HealthyChecker{
		intervalWaitService: intervalWaitService,
		stop:                stop,
		discoveryClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				DisableKeepAlives:   false,
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
			},
			// the request should happen quickly.
			Timeout: 5 * time.Second,
		},
	}
	_, _, hc.svcName = util.GetServiceCommonName()

	return hc
}

// HealthyChecker check health of the service.
type HealthyChecker struct {
	svcName             string
	intervalWaitService time.Duration
	stop                <-chan struct{}
	discoveryClient     *http.Client
}

// CheckHealthBlock block until the service is healthy or stop signal.
func (hc *HealthyChecker) CheckHealthBlock() {
	ttick := time.Tick(hc.intervalWaitService)
WAITFORSERVICE:
	for {
		if err := hc.waitForService(hc.discoveryClient, 3); err != nil {
			klog.Errorf("fail wating for svc: https://%s err:%v", hc.svcName, err)
		} else {
			klog.Infof("svc: https://%s started", hc.svcName)
			break

		}
		select {
		case <-ttick:
		case <-hc.stop:
			klog.Info("HealthyChecker.CheckHealthBlock exit with stop siganl")
			break WAITFORSERVICE
		}
	}
}

//CheckHealth signal when service is not healthy
func (hc *HealthyChecker) CheckHealth(interval time.Duration) <-chan struct{} {
	notheathy := make(chan struct{})

	go func() {
		klog.Infof("HealthChecker started with check interval:%v", interval)
		ct := time.Tick(interval)
	CKECKLOOP:
		for {
			select {
			case <-ct:
				err := hc.waitForService(hc.discoveryClient, 1)
				if err != nil {
					klog.Errorf("HealthChecker fail wating for svc: https://%s err:%v", hc.svcName, err)
					notheathy <- struct{}{}
					// Will go to CheckHealthBlock(), start again when service is not healthy
					break CKECKLOOP
				}
			case <-hc.stop:
				break CKECKLOOP
			}
		}
		klog.Infof("HealthChecker stopped")
	}()
	return notheathy
}

func (hc *HealthyChecker) waitForService(discoveryClient *http.Client, worker int) error {
	resultChan := make(chan error, worker)
	urlstr := fmt.Sprintf("https://%s/health?node=%s", hc.svcName, os.Getenv("NODENAME"))

	for i := 1; i <= worker; i++ {
		go func(worker int) {
			stime := time.Now()
			errorChan := make(chan error, 1)
			go func() {
				klog.V(9).Infof("worker%d start ", worker)
				resp, err := discoveryClient.Get(urlstr)
				if resp != nil {
					// we should always been in the 200s or 300s
					if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
						errorChan <- fmt.Errorf("bad status from %v: %v", urlstr, resp.StatusCode)
						return
					}
				}

				errorChan <- err
				klog.V(9).Infof("worker%d response end", worker)

			}()

			select {
			case val := <-errorChan:
				resultChan <- val
			case <-time.After(6 * time.Second):
				resultChan <- fmt.Errorf("time out waiting for svc to reply")
			}
			klog.V(9).Infof("worker%d end with time: %v", worker, time.Since(stime).Seconds())
		}(i)

	}

	var lasterror error
	for i := 0; i < worker; i++ {
		lasterror = <-resultChan
		if lasterror == nil {
			//if one goroutine success then success
			break
		}
	}

	return lasterror
}
