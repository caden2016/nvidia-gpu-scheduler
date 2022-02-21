// generate the common service name from Env
// (when install with helm will populate automatically) or default value in options.

package metadata

import (
	"fmt"
	"os"
	"strings"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
)

var (
	metadataName      string
	metadataNamespace string
	svcName           string
)

func init() {
	metadataName = os.Getenv("MetadataName")
	metadataNamespace = os.Getenv("MetadataNamespace")
	if metadataName == "" || metadataNamespace == "" {
		metadataName = options.ServiceName
		metadataNamespace = options.ServiceNamespace
	} else {
		//pod name like test-nvidia-gpu-scheduler-0 or test-nvidia-gpu-scheduler-facmx
		idx := strings.LastIndex(metadataName, "-")
		metadataName = metadataName[:idx]
	}
	svcName = fmt.Sprintf("%s.%s.svc", metadataName, metadataNamespace)
}

// ServiceName return service name to be used. if not, value in options is used.
func ServiceName() string {
	return svcName
}

// MetadataName return metadata name to be used. if not, value in options is used.
func MetadataName() string {
	return metadataName
}

// MetadataNamespace return metadata namespace to be used. if not, value in options is used.
func MetadataNamespace() string {
	return metadataNamespace
}
