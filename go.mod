module github.com/caden2016/nvidia-gpu-scheduler

go 1.17

require (
	github.com/NVIDIA/go-nvml v0.11.1-0
	github.com/gorilla/mux v1.8.0
	github.com/openkruise/kruise v1.0.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.9.0
	golang.org/x/sys v0.0.0-20210823070655-63515b42dcdf
	google.golang.org/grpc v1.40.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/apiserver v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.0.0
	k8s.io/kube-scheduler v0.22.4
	k8s.io/kubelet v0.22.4
	sigs.k8s.io/controller-runtime v0.10.3

)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.63.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/component-base v0.22.4 // indirect
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c // indirect
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

// Replace to match K8s 1.22.4
replace (
	k8s.io/api => k8s.io/api v0.22.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.4
	k8s.io/apiserver => k8s.io/apiserver v0.22.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.4
	k8s.io/client-go => k8s.io/client-go v0.22.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.4
	k8s.io/code-generator => k8s.io/code-generator v0.22.4
	k8s.io/component-base => k8s.io/component-base v0.22.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.4
	k8s.io/cri-api => k8s.io/cri-api v0.22.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.4
	k8s.io/kubectl => k8s.io/kubectl v0.22.4
	k8s.io/kubelet => k8s.io/kubelet v0.22.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.4
	k8s.io/metrics => k8s.io/metrics v0.22.4
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.4
)
