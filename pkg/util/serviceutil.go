package util

import (
	"fmt"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"os"
	"strings"
)

// GetServiceCommonName generate the common service name from Env
// (when install with helm will populate automatically) or default value in options.
func GetServiceCommonName() (metadataName, metadataNamespace, svcName string) {
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
	return
}
