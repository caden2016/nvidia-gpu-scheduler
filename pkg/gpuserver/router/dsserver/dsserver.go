package dsserver

import (
	"encoding/json"
	. "github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router"
	"k8s.io/klog"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"
)

//
type gpudsRouter struct {
	routes     []router.Route
	controller *controller.ServerController
}

func NewRouter(c *controller.ServerController) router.Router {
	r := &gpudsRouter{controller: c}
	r.initRoutes()
	return r
}

// Routes returns the available routes to the gpudsRouter.
func (gr *gpudsRouter) Routes() []router.Route {
	return gr.routes
}

// initRoutes initializes the routes in gpudsRouter.
func (gr *gpudsRouter) initRoutes() {
	gr.routes = []router.Route{
		router.NewGetRoute("/health", gr.getHealthHandler),
		router.NewPostRoute("/hostgpuinfo", gr.postHostgpuinfoHandler),
		router.NewPostRoute("/"+options.RESOURCES, gr.postPodresourcesHandler),
	}
}

func (gr *gpudsRouter) DumpRoutes() {
	klog.Infof("GpuDS server routes initialized.")
	for _, r := range gr.routes {
		fn := runtime.FuncForPC(reflect.ValueOf(r.Handler()).Pointer()).Name()
		fns := strings.Split(fn, ".")
		klog.Infof("%-5s%-50s\tfunc %s %s", r.Method(), r.Path(), fns[len(fns)-2], fns[len(fns)-1])
	}
}

func (gr *gpudsRouter) postHostgpuinfoHandler(w http.ResponseWriter, r *http.Request) {
	klog.V(4).Infof("get hostgpuinfo from data from exporter: %s", r.RemoteAddr)
	jdecoder := json.NewDecoder(r.Body)
	ngi := &NodeGpuInfo{}
	if err := jdecoder.Decode(ngi); err == nil {
		ngi.ReportTime = time.Now()
		klog.V(4).Infof("NodeGpuInfo:%#v", *ngi)
		gr.controller.GetNodeGpuInfoMap().SetNodeGpuInfo(ngi.NodeName, ngi)
	}
	w.WriteHeader(http.StatusOK)
}

func (gr *gpudsRouter) getHealthHandler(w http.ResponseWriter, r *http.Request) {
	node := r.URL.Query().Get("node")
	klog.V(4).Infof("notify health node:%v", node)
	gr.controller.NHC.NotifyHealth(node)
	w.WriteHeader(http.StatusOK)
}

func (gr *gpudsRouter) postPodresourcesHandler(w http.ResponseWriter, r *http.Request) {
	klog.V(4).Infof("get podresources from data from exporter: %s", r.RemoteAddr)
	jdecoder := json.NewDecoder(r.Body)
	prupdate := &PodResourceUpdate{}
	if err := jdecoder.Decode(prupdate); err == nil {
		// filter pod and container which devices is not nil and store in podresourcesIndex
		gr.controller.SendToStore(prupdate)
	}
	w.WriteHeader(http.StatusOK)
}
