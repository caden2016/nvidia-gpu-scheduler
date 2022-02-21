package framework

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
)

type Status struct {
	Accepted bool
	Err      error
}

// PluginToHostPriorityList declares a map from plugin name to its extenderv1.HostPriorityList.
type PluginToHostPriorityList map[string]extenderv1.HostPriorityList

// Plugin is the parent type for all the scheduling framework plugins.
type Plugin interface {
	Name() string
}

// FilterPlugin is an interface for Filter plugins. These plugins are called at the
// filter extension point for filtering out hosts that cannot run a pod.
type FilterPlugin interface {
	Plugin
	// Filter is called by the scheduling framework.
	Filter(ctx context.Context, pod *corev1.Pod, node string) *Status
}

// ScorePlugin is an interface that must be implemented by "Score" plugins to rank
// nodes that passed the filtering phase.
type ScorePlugin interface {
	Plugin
	// Score is called on each filtered node. It must return success and an integer
	// indicating the rank of the node.
	Score(ctx context.Context, pod *corev1.Pod, node string) (int64, *Status)
}

type PluginsRunner interface {
	RunFilterPlugins(ctx context.Context, pod *corev1.Pod, node string) *Status

	RunScorePlugins(ctx context.Context, pod *corev1.Pod, nodes []string, parallelism int) extenderv1.HostPriorityList
}

type Framework interface {
	PluginsRunner
}
