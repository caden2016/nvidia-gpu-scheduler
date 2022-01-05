package metricserver

import (
	"encoding/json"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/apis"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router"
	serverutil "github.com/caden2016/nvidia-gpu-scheduler/pkg/util/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schemeruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/klog"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"
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
			Name:         options.RESOURCES,
			SingularName: options.RESOURCE,
			Namespaced:   false,
			Kind:         options.KIND,
			Verbs:        metav1.Verbs([]string{"list", "watch"}),
		},
		metav1.APIResource{
			Name:         options.GPURESOURCES,
			SingularName: options.GPURESOURCE,
			Namespaced:   false,
			Kind:         options.GPUKIND,
			Verbs:        metav1.Verbs([]string{"list"}),
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
		router.NewGetRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION}...), mr.getVersionHandler),
		router.NewGetRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.RESOURCES}...), mr.getResourceHandler),
		router.NewGetRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.GPURESOURCES}...), mr.getGpuInfoHandler),
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

func (mr *metricsRouter) getVersionHandler(w http.ResponseWriter, r *http.Request) {
	versionHandler.ServeHTTP(w, r)
}

func (mr *metricsRouter) getResourceHandler(w http.ResponseWriter, r *http.Request) {

	// Watch will list first then continue watch.
	if r.URL.Query().Get("watch") == "true" {
		watchResultChan := make(chan *apis.Event, 10)
		watchChan := make(chan *PodResourceUpdate)

		watch_uuid := mr.controller.AddWatcher(watchChan)
		defer mr.controller.DelWatcher(watch_uuid)
		klog.Infof("get watch uuid:%s", watch_uuid)

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
		dataList := <-mr.controller.GetFromStore()
		go func() {
			for _, apipr := range serverutil.PodResourcesDetailToPodResource(dataList) {
				watchResultChan <- &apis.Event{Type: apis.Synced, Object: apipr}
			}

			//watchChan not stop until client disconnect.
			for pru := range watchChan {
				for _, apipr := range serverutil.PodResourcesDetailToPodResource(pru.PodResourcesSYNC) {
					watchResultChan <- &apis.Event{Type: apis.Synced, Object: apipr}
				}

				for _, apipr := range pru.PodResourcesDEL {
					watchResultChan <- &apis.Event{
						Type: apis.Deleted,
						Object: apis.NewPodResource(
							&apis.PodResourceSpec{
								Name:             apipr.Name,
								Namespace:        apipr.Namespace,
								ContainerDevices: nil,
							},
							&apis.PodResourceStatus{
								LastChangedTime: time.Now().Format(time.RFC3339),
							}),
					}
				}
			}
			// When ServerController clean all connections, we need to exit the LOOP goroutine.
			close(watchResultChan)
			klog.V(4).Infof("watch uuid:%s watchChan receive close signal and will be removed", watch_uuid)
		}()

	LOOP:
		for {
			select {
			case <-r.Context().Done():
				klog.Infof("watch uuid:%s connection end from the client", watch_uuid)
				close(watchChan)
				break LOOP

			case wevent, ok := <-watchResultChan:
				if ok {
					if err := jsonenc.Encode(wevent); err != nil {
						klog.Errorf("watch uuid:%s jsonenc.Encode err:%v", watch_uuid, err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					flusher.Flush()
					klog.V(4).Infof("watch uuid:%s data send", watch_uuid)
				} else {
					//resultChan is closed
					klog.Infof("watch uuid:%s closed", watch_uuid)
					break LOOP
				}
			}
		}

		klog.Infof("watch uuid:%s exit", watch_uuid)
		return
	}

	// Get just list the result.
	select {
	case data := <-mr.controller.GetFromStore():
		apiprl := serverutil.PodResourcesDetailToPodResource(data)
		prListOut := apis.NewList(len(apiprl))
		for _, apipr := range apiprl {
			prListOut.Items = append(prListOut.Items, apipr)
		}

		err := serverutil.WriteJSON(w, http.StatusOK, prListOut)
		if err != nil {
			klog.Errorf("WriteJSON Error: %v", err)
		}
		return
	case <-time.After(5 * time.Second):
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
}

func (mr *metricsRouter) getGpuInfoHandler(w http.ResponseWriter, _ *http.Request) {
	ngiList := mr.controller.GetNodeGpuInfoMap().GetAllNodeGpuInfo()
	ngiListOut := apis.NewList(len(ngiList))
	for _, ngi := range ngiList {
		nodeDeviceInUse := mr.controller.GetNodeGpuInfoMap().GetNodeDeviceInUse(ngi.NodeName)
		nodestatus := mr.controller.NHC.GetNodeStatus(ngi.NodeName)
		ngiListOut.Items = append(ngiListOut.Items, apis.NewGpuInfo(&apis.GpuInfoSpec{
			NodeName:        ngi.NodeName,
			GpuInfos:        ngi.GpuInfos,
			Models:          serverutil.MapSetToList(ngi.Models),
			ReportTime:      ngi.ReportTime,
			NodeDeviceInUse: nodeDeviceInUse.List(),
		}, serverutil.NodeStatusToGpuInfoStatus(nodestatus)))
	}
	err := serverutil.WriteJSON(w, http.StatusOK, ngiListOut)
	if err != nil {
		klog.Errorf("WriteJSON Error: %v", err)
	}
}
