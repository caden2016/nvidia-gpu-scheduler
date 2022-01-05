package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/openkruise/kruise/pkg/webhook/util/generator"
	"gopkg.in/yaml.v2"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"net"
	"net/http"
	"os"
	"time"
)

// GetTlsConfig get tls config with server config certs files.
func GetTlsConfig(certs *generator.Artifacts) (*tls.Config, error) {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certs.CACert)
	Crt, err := tls.X509KeyPair(certs.Cert, certs.Key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{Crt},
	}, nil
}

// GetListener get standard net Listener with server config and tls config.
func GetListener(sflags *options.MetricsPodResourceFlags, cfg *tls.Config) (listener net.Listener, err error) {
	listener, err = tls.Listen("tcp", fmt.Sprintf("%s:%d", sflags.BindAddress, sflags.BindPort), cfg)
	return
}

// WriteJSON send back http stream, encode the value v with standard json encoding.
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func LogOrWriteConfig(fileName string, sflags interface{}) error {
	outyaml, err := yaml.Marshal(sflags)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(outyaml)
	if klog.V(2) {
		klog.Info("Using server config", "\n-------------------------Configuration File Contents Start Here---------------------- \n", buf.String(), "\n------------------------------------Configuration File Contents End Here---------------------------------\n")
	}

	if len(fileName) > 0 {
		configFile, err := os.Create(fileName)
		if err != nil {
			return err
		}
		defer configFile.Close()
		if _, err := io.Copy(configFile, buf); err != nil {
			return err
		}
		klog.Infof("Wrote configuration to the file: %s", fileName)
		os.Exit(0)
	}
	return nil
}

func DumpConfig(sflags interface{}) error {
	outyaml, err := yaml.Marshal(sflags)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(outyaml)
	klog.Info("Using server config", "\n-------------------------Configuration File Contents Start Here---------------------- \n", buf.String(), "\n------------------------------------Configuration File Contents End Here---------------------------------\n")
	return nil
}

func GetKubeAndAggregatorClientset() (kubernetes.Interface, clientset.Interface, error) {
	restconf, err := rest.InClusterConfig()
	//restconf, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	if err != nil {
		return nil, nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(restconf)
	if err != nil {
		return nil, nil, err
	}

	aggregatorClient, err := clientset.NewForConfig(restconf)
	if err != nil {
		return nil, nil, err
	}

	return kubeClient, aggregatorClient, nil
}

// EnsureAPIService ensure the apiservice.spec.caBundle is fresh
func EnsureAPIService(aggregatorClient clientset.Interface, cacert []byte) error {
	aggctx, cancel := context.WithTimeout(context.TODO(), time.Second*2)
	defer cancel()
	apiserviceName := options.APIVERSION + "." + options.APIGROUP
	base64str := base64.StdEncoding.EncodeToString(cacert)
	patchBytes := []byte(fmt.Sprintf(`{"spec":{"caBundle":"%s","insecureSkipTLSVerify": false}}`, string(base64str)))
	patchOpt := metav1.PatchOptions{FieldManager: options.CERT_Secret_Name}
	_, err := aggregatorClient.ApiregistrationV1().APIServices().Patch(aggctx, apiserviceName, types.StrategicMergePatchType, patchBytes, patchOpt)
	return err
}
