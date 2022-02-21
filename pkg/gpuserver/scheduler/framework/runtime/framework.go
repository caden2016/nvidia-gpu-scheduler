package runtime

import (
	"context"
	"reflect"

	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
)

func NewFramework(r Registry) (framework.Framework, error) {
	fw := &frameworkImpl{registry: r}
	pluginsList := make([]framework.Plugin, 0, len(r))

	for _, factory := range r {
		plugin, err := factory()
		if err != nil {
			return nil, err
		}
		pluginsList = append(pluginsList, plugin)
	}

	// Add ExtensionPoints to correlated plugin list in the framework.
	for _, ep := range fw.getExtensionPoints() {
		if err := addPluginList(ep, pluginsList); err != nil {
			return nil, err
		}
	}

	return fw, nil
}

func addPluginList(pluginList interface{}, pluginsList []framework.Plugin) error {
	plugins := reflect.ValueOf(pluginList).Elem()
	pluginType := plugins.Type().Elem()

	for _, pl := range pluginsList {
		plName := pl.Name()
		if reflect.TypeOf(pl).Implements(pluginType) {
			newPlugins := reflect.Append(plugins, reflect.ValueOf(pl))
			plugins.Set(newPlugins)
			klog.Infof("plugin %q implements %s plugin", plName, pluginType.Name())
		}
	}
	return nil
}

// frameworkImpl is the component responsible for initializing and running scheduler
// plugins.
type frameworkImpl struct {
	registry      Registry
	filterPlugins []framework.FilterPlugin
	scorePlugins  []framework.ScorePlugin
}

func (f *frameworkImpl) getExtensionPoints() []interface{} {
	return []interface{}{
		&f.scorePlugins,
		&f.filterPlugins,
	}
}

func (f *frameworkImpl) RunFilterPlugins(ctx context.Context, pod *corev1.Pod, node string) *framework.Status {
	for _, filter := range f.filterPlugins {
		status := filter.Filter(ctx, pod, node)
		if !status.Accepted {
			klog.Infof("Plugin[%s].Filter refused with Error: %v", filter.Name(), status.Err)
			return status
		}
	}

	return &framework.Status{Accepted: true}
}

func (f *frameworkImpl) RunScorePlugins(ctx context.Context, pod *corev1.Pod, nodes []string, parallelism int) extenderv1.HostPriorityList {
	hpList := make(extenderv1.HostPriorityList, 0, len(nodes))
	pluginToNodeScores := make(framework.PluginToHostPriorityList, len(f.scorePlugins))
	for _, pl := range f.scorePlugins {
		pluginToNodeScores[pl.Name()] = make(extenderv1.HostPriorityList, len(nodes))
	}

	workqueue.ParallelizeUntil(ctx, parallelism, len(nodes), func(i int) {
		for _, pl := range f.scorePlugins {
			score, status := pl.Score(ctx, pod, nodes[i])
			if !status.Accepted {
				klog.Infof("Plugin[%s].Score %v", pl.Name(), status.Err)
			}
			pluginToNodeScores[pl.Name()][i] = extenderv1.HostPriority{
				Host:  nodes[i],
				Score: score,
			}
		}
	})

	// summarize scores
	for i := range nodes {
		hpList = append(hpList, extenderv1.HostPriority{Host: nodes[i], Score: 0})
		for j := range pluginToNodeScores {
			klog.V(4).Infof("Plugin:%s node:%s score:%d", j, nodes[i], pluginToNodeScores[j][i].Score)
			hpList[i].Score += pluginToNodeScores[j][i].Score
		}
	}

	return hpList
}
