package client

import (
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	crdClientset "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	devicev1alpha2 "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned/typed/devices/v1alpha2"
	syncv1alpha1 "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned/typed/reliablesyncs/v1alpha1"
	cloudcoreConfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type KubeEdgeClient interface {
	kubernetes.Interface
	ClusterObjectSync() syncv1alpha1.ClusterObjectSyncInterface
	ObjectSync(namespace string) syncv1alpha1.ObjectSyncInterface
	Device(namespace string) devicev1alpha2.DeviceInterface
	DeviceModel(namespace string) devicev1alpha2.DeviceModelInterface
	ReliablesyncsRestClient() restclient.Interface
	DevicesRestClient() restclient.Interface
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
	kubeConfig.ContentType = runtime.ContentTypeJSON
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	crdClient := crdClientset.NewForConfigOrDie(kubeConfig)
	once.Do(func() {
		keClient = &kubeedgeClient{
			Clientset: kubeClient,
			crdClient: crdClient,
		}
	})
}

func GetKubeEdgeClient() KubeEdgeClient {
	return keClient
}

type kubeedgeClient struct {
	*kubernetes.Clientset
	crdClient *crdClientset.Clientset
}

func (kec *kubeedgeClient) ReliablesyncsRestClient() restclient.Interface {
	return kec.crdClient.ReliablesyncsV1alpha1().RESTClient()
}

func (kec *kubeedgeClient) DevicesRestClient() restclient.Interface {
	return kec.crdClient.DevicesV1alpha2().RESTClient()
}

func (kec *kubeedgeClient) ClusterObjectSync() syncv1alpha1.ClusterObjectSyncInterface {
	return kec.crdClient.ReliablesyncsV1alpha1().ClusterObjectSyncs()
}

func (kec *kubeedgeClient) ObjectSync(namespace string) syncv1alpha1.ObjectSyncInterface {
	return kec.crdClient.ReliablesyncsV1alpha1().ObjectSyncs(namespace)
}

func (kec *kubeedgeClient) Device(namespace string) devicev1alpha2.DeviceInterface {
	return kec.crdClient.DevicesV1alpha2().Devices(namespace)
}

func (kec *kubeedgeClient) DeviceModel(namespace string) devicev1alpha2.DeviceModelInterface {
	return kec.crdClient.DevicesV1alpha2().DeviceModels(namespace)
}
