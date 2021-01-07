package client

import (
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	crdClientset "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	devicev1alpha2 "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned/typed/devices/v1alpha2"
	syncv1alpha1 "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned/typed/reliablesyncs/v1alpha1"
	cloudcoreConfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type KubeEdgeClient interface {
	kubernetes.Interface
	crdClientset.Interface
}

var keClient KubeEdgeClient
var once sync.Once

func InitKubeEdgeClient(config *cloudcoreConfig.KubeAPIConfig) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Master,
		config.KubeConfig)
	if err != nil {
		klog.Errorf("Failed to build config, err: %v", err)
		os.Exit(1)
	}
	kubeConfig.QPS = float32(config.QPS)
	kubeConfig.Burst = int(config.Burst)
	kubeConfig.ContentType = runtime.ContentTypeProtobuf
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	crdKubeConfig := rest.CopyConfig(kubeConfig)
	crdKubeConfig.ContentType = runtime.ContentTypeJSON
	crdClient := crdClientset.NewForConfigOrDie(crdKubeConfig)
	once.Do(func() {
		keClient = &kubeEdgeClient{
			Clientset: kubeClient,
			crdClient: crdClient,
		}
	})
}

func GetKubeEdgeClient() KubeEdgeClient {
	return keClient
}

type kubeEdgeClient struct {
	*kubernetes.Clientset
	crdClient *crdClientset.Clientset
}

func (kec *kubeEdgeClient) DevicesV1alpha2() devicev1alpha2.DevicesV1alpha2Interface {
	return kec.crdClient.DevicesV1alpha2()
}

func (kec *kubeEdgeClient) ReliablesyncsV1alpha1() syncv1alpha1.ReliablesyncsV1alpha1Interface {
	return kec.crdClient.ReliablesyncsV1alpha1()
}
