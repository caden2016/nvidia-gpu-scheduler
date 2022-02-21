package metricserver

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strings"

	gpunodev1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpunode/v1"
	gpupodv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server/watcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schemeruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/klog"
)

var (
	Scheme         = schemeruntime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	versionHandler *discovery.APIVersionHandler
)

func init() {

	// if you modify this, make sure you update the crEncoder
	unversionedVersion := schema.GroupVersion{Group: "", Version: "v1"}
	unversionedTypes := []schemeruntime.Object{
		&metav1.Status{},
		&metav1.WatchEvent{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	}
	Scheme.AddUnversionedTypes(unversionedVersion, unversionedTypes...)
	apiResourcesForDiscovery := make([]metav1.APIResource, 0, 2)
	apiResourcesForDiscovery = append(apiResourcesForDiscovery,
		metav1.APIResource{
			Name:         options.RESOURCES_GPUPOD,
			SingularName: options.RESOURCE_GPUPOD,
			Namespaced:   false,
			Kind:         options.KIND_GPUPOD,
			Verbs:        []string{"list", "watch"},
		},
		metav1.APIResource{
			Name:         options.RESOURCES_GPUNODE,
			SingularName: options.RESOURCE_GPUNODE,
			Namespaced:   false,
			Kind:         options.KIND_GPUNODE,
			Verbs:        []string{"list", "watch"},
		},
	)

	versionHandler = discovery.NewAPIVersionHandler(Codecs, schema.GroupVersion{Group: options.APIGROUP, Version: options.APIVERSION}, discovery.APIResourceListerFunc(func() []metav1.APIResource {
		return apiResourcesForDiscovery
	}))

}

type metricsRouter struct {
	routes     []router.Route
	controller *controller.ServerController
}

func NewRouter(c *controller.ServerController) router.Router {
	r := &metricsRouter{controller: c}
	r.initRoutes()
	return r
}

// Routes returns the available routes to the metricsRouter.
func (mr *metricsRouter) Routes() []router.Route {
	return mr.routes
}

// initRoutes initializes the routes in metricsRouter.
func (mr *metricsRouter) initRoutes() {
	mr.routes = []router.Route{
		router.NewGetRoute("/health", mr.getHealthHandler),
		router.NewGetRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION}...), mr.getVersionHandler),
		router.NewGetRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.RESOURCES_GPUPOD}...), mr.getGpuPodHandler),
		router.NewGetRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.RESOURCES_GPUNODE}...), mr.getGpuNodeHandler),
	}
}

func (mr *metricsRouter) DumpRoutes() {
	klog.Infof("Metrics server routes initialized.")
	for _, r := range mr.routes {
		fn := runtime.FuncForPC(reflect.ValueOf(r.Handler()).Pointer()).Name()
		fns := strings.Split(fn, ".")
		klog.Infof("%-5s%-50s\tfunc %s %s", r.Method(), r.Path(), fns[len(fns)-2], fns[len(fns)-1])
	}
}

func (mr *metricsRouter) getHealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (mr *metricsRouter) getVersionHandler(w http.ResponseWriter, r *http.Request) {
	versionHandler.ServeHTTP(w, r)
}

func (mr *metricsRouter) getGpuPodHandler(w http.ResponseWriter, r *http.Request) {
	// Watch will list first then continue watch.
	if r.URL.Query().Get("watch") == "true" {
		watchResultChan := make(chan *watcher.GpuPodEvent, 10)
		watchChan := make(chan interface{}, 1)

		watchUuid := watcher.GpuPodWatcher.AddWatcher(watchChan)
		klog.Infof("get watch uuid:%s", watchUuid)

		flusher, ok := w.(http.Flusher)
		if !ok {
			klog.Errorf("w is not a http.Flusher")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Del("Content-Length")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		flusher.Flush() //chunked need return success head first

		jsonenc := json.NewEncoder(w)

		// add list result before watch signal arrives.

		gpList := &gpupodv1.GpuPodList{}
		if err := mr.controller.GpuMgrClient.List(context.TODO(), gpList); err != nil {
			klog.Errorf("watch uuid:%s GpuMgrClient.List err:%v", watchUuid, err)
		}

		go func() {
			for _, gp := range gpList.Items {
				watchResultChan <- watcher.NewGpuPodEvent(&gp, watcher.Synced)
			}

			//watchChan not stop until client disconnect.
			for gpe := range watchChan {
				watchResultChan <- gpe.(*watcher.GpuPodEvent)
			}
			// When ServerController clean all connections, we need to exit the LOOP goroutine.
			close(watchResultChan)
			klog.V(4).Infof("watch uuid:%s watchChan receive close signal and will be removed", watchUuid)
		}()

	LOOP:
		for {
			select {
			case <-r.Context().Done():
				klog.Infof("watch uuid:%s connection end from the client", watchUuid)
				watcher.GpuPodWatcher.DelWatcher(watchUuid)
				close(watchChan)
				break LOOP

			case wevent, ok := <-watchResultChan:
				if ok {
					if err := jsonenc.Encode(wevent); err != nil {
						klog.Errorf("watch uuid:%s jsonenc.Encode err:%v", watchUuid, err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					flusher.Flush()
					klog.V(4).Infof("watch uuid:%s data send", watchUuid)
				} else {
					//resultChan is closed
					klog.Infof("watch uuid:%s closed", watchUuid)
					break LOOP
				}
			}
		}

		klog.Infof("watch uuid:%s exit", watchUuid)
		return
	}

	// Get just list the result.
	gpList := &gpupodv1.GpuPodList{}
	if err := mr.controller.GpuMgrClient.List(context.TODO(), gpList); err != nil {
		klog.Errorf("GpuMgrClient.List err:%v", err)
	}
	serverutil.WriteJSON(w, http.StatusOK, gpList)
}

func (mr *metricsRouter) getGpuNodeHandler(w http.ResponseWriter, r *http.Request) {

	// Watch will list first then continue watch.
	if r.URL.Query().Get("watch") == "true" {
		watchResultChan := make(chan *watcher.GpuNodeEvent, 10)
		watchChan := make(chan interface{}, 1)

		watchUuid := watcher.GpuNodeWatcher.AddWatcher(watchChan)
		klog.Infof("get watch uuid:%s", watchUuid)

		flusher, ok := w.(http.Flusher)
		if !ok {
			klog.Errorf("w is not a http.Flusher")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Del("Content-Length")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		flusher.Flush() //chunked need return success head first

		jsonenc := json.NewEncoder(w)

		// add list result before watch signal arrives.

		gpnList := &gpunodev1.GpuNodeList{}
		if err := mr.controller.GpuMgrClient.List(context.TODO(), gpnList); err != nil {
			klog.Errorf("watch uuid:%s GpuMgrClient.List err:%v", watchUuid, err)
		}

		go func() {
			for _, gpn := range gpnList.Items {
				watchResultChan <- watcher.NewGpuNodeEvent(&gpn, watcher.Synced)
			}

			//watchChan not stop until client disconnect.
			for gpne := range watchChan {
				watchResultChan <- gpne.(*watcher.GpuNodeEvent)
			}
			// When ServerController clean all connections, we need to exit the LOOP goroutine.
			close(watchResultChan)
			klog.V(4).Infof("watch uuid:%s watchChan receive close signal and will be removed", watchUuid)
		}()

	LOOP:
		for {
			select {
			case <-r.Context().Done():
				klog.Infof("watch uuid:%s connection end from the client", watchUuid)
				watcher.GpuNodeWatcher.DelWatcher(watchUuid)
				close(watchChan)
				break LOOP

			case wevent, ok := <-watchResultChan:
				if ok {
					if err := jsonenc.Encode(wevent); err != nil {
						klog.Errorf("watch uuid:%s jsonenc.Encode err:%v", watchUuid, err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					flusher.Flush()
					klog.V(4).Infof("watch uuid:%s data send", watchUuid)
				} else {
					//resultChan is closed
					klog.Infof("watch uuid:%s closed", watchUuid)
					break LOOP
				}
			}
		}

		klog.Infof("watch uuid:%s exit", watchUuid)
		return
	}

	// Get just list the result.
	gpnList := &gpunodev1.GpuNodeList{}
	if err := mr.controller.GpuMgrClient.List(context.TODO(), gpnList); err != nil {
		klog.Errorf("GpuMgrClient.List err:%v", err)
	}
	serverutil.WriteJSON(w, http.StatusOK, gpnList)
}
