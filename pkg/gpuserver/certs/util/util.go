package util

import (
	"fmt"
	"io/ioutil"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/util/info/metadata"
	"github.com/openkruise/kruise/pkg/webhook/util/generator"
	"github.com/openkruise/kruise/pkg/webhook/util/writer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func InitializeCert(kubeClient kubernetes.Interface) (*generator.Artifacts, error) {
	name, namespace, dnsName := metadata.MetadataName(), metadata.MetadataNamespace(), metadata.ServiceName()
	klog.Infof("generating cert with dnsName:%s", dnsName)

	certWriter, err := writer.NewSecretCertWriter(writer.SecretCertWriterOptions{
		Clientset: kubeClient,
		Secret:    &types.NamespacedName{Namespace: namespace, Name: name},
	})
	if err != nil {
		return nil, fmt.Errorf("NewSecretCertWriter: %v", err)
	}

	certs, changed, err := certWriter.EnsureCert(dnsName)
	if err != nil {
		return nil, fmt.Errorf("EnsureCert: %v", err)
	}

	if err := writer.WriteCertsToDir(options.CERTDIR, certs); err != nil {
		return nil, fmt.Errorf("EnsureCert failed to write certs to dir")
	}

	klog.Infof("EnsureCert write certs to dir success cert file changed: %t", changed)
	return certs, nil
}

func InitializeCertWithFile(tlsconfig *options.TLSCONFIG) (*generator.Artifacts, error) {
	caCertBytes, err := ioutil.ReadFile(tlsconfig.CACert)
	if err != nil {
		return nil, err
	}
	certBytes, err := ioutil.ReadFile(tlsconfig.Cert)
	if err != nil {
		return nil, err
	}
	keyBytes, err := ioutil.ReadFile(tlsconfig.Key)
	if err != nil {
		return nil, err
	}
	return &generator.Artifacts{
		CACert: caCertBytes,
		Cert:   certBytes,
		Key:    keyBytes,
	}, nil
}
