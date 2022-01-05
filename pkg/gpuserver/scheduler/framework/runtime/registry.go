package runtime

import (
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/scheduler/framework"
)

// PluginFactory is a function that builds a plugin.
type PluginFactory = func() (framework.Plugin, error)

type Registry map[string]PluginFactory
