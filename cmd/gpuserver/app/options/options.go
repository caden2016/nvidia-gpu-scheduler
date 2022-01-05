package options

type MetricsPodResourceFlags struct {
	BindAddress     string          `mapstructure:"bind-address" yaml:"bind-address,omitempty"`
	BindPort        int             `mapstructure:"secure-port" yaml:"secure-port,omitempty"`
	TLSAuto         bool            `mapstructure:"tls-auto" yaml:"tls-autos"`
	TLSConfig       TLSCONFIG       `mapstructure:"tls-config" yaml:"tls-config,omitempty"`
	WriteConfigTo   string          `mapstructure:"write-config-to" yaml:"-"`
	EnableScheduler bool            `mapstructure:"enable-scheduler" yaml:"enable-scheduler"`
	Scheduler       SchedulerConfig `mapstructure:"scheduler" yaml:"scheduler"`
}

type TLSCONFIG struct {
	CACert string `mapstructure:"tls-ca-file" yaml:"tls-ca-file,omitempty"`
	Cert   string `mapstructure:"tls-cert-file" yaml:"tls-cert-file,omitempty"`
	Key    string `mapstructure:"tls-private-key-file" yaml:"tls-private-key-file,omitempty"`
}

type SchedulerConfig struct {
	Parallelism int `mapstructure:"parallelism" yaml:"parallelism"`
}
