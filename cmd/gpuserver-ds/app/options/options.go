package options

import "time"

type MetricsPodResourceDSFlags struct {
	WriteConfigTo             string        `mapstructure:"write-config-to" yaml:"-"`
	LocalPodResourcesEndpoint string        `mapstructure:"localPodResourcesEndpoint" yaml:"localPodResourcesEndpoint,omitempty"`
	IntervalWaitService       time.Duration `mapstructure:"interval-wait-service" yaml:"interval-wait-service"`
}
