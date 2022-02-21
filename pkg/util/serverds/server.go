package serverds

import (
	"context"
	"time"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	serveroptions "github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/info/metadata"
	"github.com/openkruise/kruise/pkg/webhook/util/writer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// EnsureGetCaFromSecrets block until get the ca from secret
func EnsureGetCaFromSecrets(kubeclient kubernetes.Interface) (cacert []byte) {
	var secret *corev1.Secret
	var err error

	name, namespace := metadata.MetadataName(), metadata.MetadataNamespace()
	parentcxt, cancel := context.WithCancel(context.TODO())
	defer cancel()
	tc := time.Tick(options.CaFromSecret_CheckInterval)
	for range tc {
		ctx, cancelFunc := context.WithTimeout(parentcxt, time.Second*2)
		secret, err = kubeclient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		cancelFunc()
		if err != nil {
			klog.Errorf("EnsureGetCaFromSecrets: cannot get ca from secrets[%s/%s]:%v",
				serveroptions.CERT_Secret_Namespace, serveroptions.CERT_Secret_Name, err)
		} else {
			break
		}
	}

	return secret.Data[writer.CACertName]
}
