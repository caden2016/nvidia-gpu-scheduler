package options

type MetricsPodResourceDSFlags struct {
	WriteConfigTo             string `mapstructure:"write-config-to" yaml:"-"`
	LocalPodResourcesEndpoint string `mapstructure:"localPodResourcesEndpoint" yaml:"localPodResourcesEndpoint,omitempty"`
}
