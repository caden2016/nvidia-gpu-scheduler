package init

import (
	"encoding/json"
	"net/http"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router/metricserver"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router/schedulerserver"
	"github.com/gorilla/mux"
	"k8s.io/klog"
)

func RegisterRoutes(m *mux.Router, sc *controller.ServerController, enableScheduler bool) {
	allrouter := []router.Router{
		metricserver.NewRouter(sc),
	}
	if enableScheduler {
		klog.Infof("Flag enable-scheduler is enabled, register routes for scheduler server.")
		allrouter = append(allrouter, schedulerserver.NewRouter(sc))
	}

	for _, routers := range allrouter {
		for _, r := range routers.Routes() {
			m.Path(r.Path()).Methods(r.Method()).Handler(r.Handler())
		}
		routers.DumpRoutes()
	}
	m.Use(loggingMiddleware)
	m.NotFoundHandler = http.HandlerFunc(notFoundHandler)

}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(map[string]interface{}{
		"msg":  "api not found",
		"code": 404,
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		klog.V(4).Infof("M[%s] F[%s] URI[%s]", r.Method, r.RemoteAddr, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
