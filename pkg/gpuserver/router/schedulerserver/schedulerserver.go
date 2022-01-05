package schedulerserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/controller"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/router"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

type schedulerRouter struct {
	routes     []router.Route
	controller *controller.ServerController
}

func NewRouter(c *controller.ServerController) router.Router {
	r := &schedulerRouter{controller: c}
	r.initRoutes()
	return r
}

// Routes returns the available routes to the schedulerRouter.
func (sr *schedulerRouter) Routes() []router.Route {
	return sr.routes
}

// initRoutes initializes the routes in schedulerRouter.
func (sr *schedulerRouter) initRoutes() {
	sr.routes = []router.Route{
		router.NewPostRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.SCHEDULE, options.SCHEDULE_FILTER}...), sr.postScheduleFilterHandler),
		router.NewPostRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.SCHEDULE, options.SCHEDULE_PRIORITIZE}...), sr.postSchedulePrioritizeHandler),
		router.NewPostRoute(path.Join([]string{"/apis", options.APIGROUP, options.APIVERSION, options.SCHEDULE, options.SCHEDULE_PREEMPT}...), sr.postSchedulePreemptHandler),
	}
}

func (sr *schedulerRouter) DumpRoutes() {
	klog.Infof("Scheduler server routes initialized.")
	for _, r := range sr.routes {
		fn := runtime.FuncForPC(reflect.ValueOf(r.Handler()).Pointer()).Name()
		fns := strings.Split(fn, ".")
		klog.Infof("%-5s%-50s\tfunc %s %s", r.Method(), r.Path(), fns[len(fns)-2], fns[len(fns)-1])
	}
}

func (sr *schedulerRouter) postScheduleFilterHandler(w http.ResponseWriter, r *http.Request) {
	searg := &extenderv1.ExtenderArgs{}
	seresult := &extenderv1.ExtenderFilterResult{}
	jdecoder := json.NewDecoder(r.Body)
	jencoder := json.NewEncoder(w)

	if err := jdecoder.Decode(searg); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	klog.Infof("Before schedule filter pod:%s/%s, annotation:%v ExtenderArgs nodes:%v", searg.Pod.Namespace, searg.Pod.Name, searg.Pod.Annotations, *(searg.NodeNames))

	var feasibleNodesLen int32
	nodeNum := len(*searg.NodeNames)
	feasibleNodes := make([]string, nodeNum)
	checkNode := func(i int) {
		status := sr.controller.FW.RunFilterPlugins(context.TODO(), searg.Pod, sr.controller.FWNI, (*searg.NodeNames)[i])
		if status.Accepted {
			length := atomic.AddInt32(&feasibleNodesLen, 1)
			feasibleNodes[length-1] = (*searg.NodeNames)[i]
		}
	}

	workqueue.ParallelizeUntil(context.TODO(), sr.controller.GetParallelism(), nodeNum, checkNode)
	feasibleNodesFinal := feasibleNodes[:feasibleNodesLen]
	seresult.NodeNames = &feasibleNodesFinal

	klog.Infof("After schedule filter pod:%s/%s, available nodes:%v", searg.Pod.Namespace, searg.Pod.Name, *(seresult.NodeNames))
	if err := jencoder.Encode(seresult); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (sr *schedulerRouter) postSchedulePrioritizeHandler(w http.ResponseWriter, r *http.Request) {
	searg := &extenderv1.ExtenderArgs{}
	seresult := extenderv1.HostPriorityList{}
	jdecoder := json.NewDecoder(r.Body)
	jencoder := json.NewEncoder(w)
	if err := jdecoder.Decode(searg); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	klog.Infof("Before schedule prioritize pod:%s/%s, annotation:%v ExtenderArgs nodes:%v", searg.Pod.Namespace, searg.Pod.Name, searg.Pod.Annotations, *(searg.NodeNames))

	seresult = sr.controller.FW.RunScorePlugins(context.TODO(), searg.Pod, sr.controller.FWNI, *searg.NodeNames, sr.controller.GetParallelism())

	klog.Infof("After schedule prioritize pod:%s/%s, HostPriority:%v", searg.Pod.Namespace, searg.Pod.Name, seresult)
	if err := jencoder.Encode(seresult); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (sr *schedulerRouter) postSchedulePreemptHandler(w http.ResponseWriter, r *http.Request) {
	searg := &extenderv1.ExtenderPreemptionArgs{}
	seresult := &extenderv1.ExtenderPreemptionResult{}
	jdecoder := json.NewDecoder(r.Body)
	jencoder := json.NewEncoder(w)

	if err := jdecoder.Decode(searg); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	klog.Info(formatExtenderPreemptionArgs(searg))

	argNodeNames := make([]string, 0, len(searg.NodeNameToMetaVictims))
	seresult.NodeNameToMetaVictims = make(map[string]*extenderv1.MetaVictims)
	for nodeName := range searg.NodeNameToMetaVictims {
		argNodeNames = append(argNodeNames, nodeName)
	}

	var feasibleNodesLen int32
	nodeNum := len(searg.NodeNameToMetaVictims)
	feasibleNodes := make([]string, nodeNum)
	checkNode := func(i int) {
		status := sr.controller.FW.RunFilterPlugins(context.TODO(), searg.Pod, sr.controller.FWNI, argNodeNames[i])
		if status.Accepted {
			length := atomic.AddInt32(&feasibleNodesLen, 1)
			feasibleNodes[length-1] = argNodeNames[i]
		}
	}

	workqueue.ParallelizeUntil(context.TODO(), sr.controller.GetParallelism(), nodeNum, checkNode)
	feasibleNodesFinal := feasibleNodes[:feasibleNodesLen]
	for _, nodeName := range feasibleNodesFinal {
		seresult.NodeNameToMetaVictims[nodeName] = searg.NodeNameToMetaVictims[nodeName]
	}

	klog.Infof("After schedule preempt pod:%s/%s, available nodes:%v", searg.Pod.Namespace, searg.Pod.Name, seresult.NodeNameToMetaVictims)
	if err := jencoder.Encode(seresult); err != nil {
		klog.Error(err)
		w.WriteHeader(http.StatusBadRequest)
	}
}

// formatExtenderPreemptionArgs format extenderv1.ExtenderPreemptionArgs.
func formatExtenderPreemptionArgs(searg *extenderv1.ExtenderPreemptionArgs) string {
	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("Before schedule preempt pod:%s/%s annotation:%v ", searg.Pod.Namespace, searg.Pod.Name, searg.Pod.Annotations))
	for nm, nmv := range searg.NodeNameToMetaVictims {
		result.WriteString(strings.Join([]string{"nodename: ", nm, " NumPDBViolations: ", strconv.FormatInt(nmv.NumPDBViolations, 10)}, ""))
		result.WriteString(" pods:")
		for _, mp := range nmv.Pods {
			result.WriteString(mp.UID + ",")
		}
	}
	return result.String()
}
