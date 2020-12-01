package cloudhub

import (
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	crdinformerfactory "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var DoneTLSTunnelCerts = make(chan bool, 1)

type cloudHub struct {
	enable bool
}

func newCloudHub(enable bool) *cloudHub {
	return &cloudHub{
		enable: enable,
	}
}

func Register(hub *v1alpha1.CloudHub, kubeAPIConfig *v1alpha1.KubeAPIConfig) {
	hubconfig.InitConfigure(hub, kubeAPIConfig)
	core.Register(newCloudHub(hub.Enable))
}

func (a *cloudHub) Name() string {
	return modules.CloudHubModuleName
}

func (a *cloudHub) Group() string {
	return modules.CloudHubModuleGroup
}

// Enable indicates whether enable this module
func (a *cloudHub) Enable() bool {
	return a.enable
}

func (a *cloudHub) Start() {
	objectSyncController := newObjectSyncController()

	if !cache.WaitForCacheSync(beehiveContext.Done(),
		objectSyncController.ClusterObjectSyncSynced,
		objectSyncController.ObjectSyncSynced,
	) {
		klog.Errorf("unable to sync caches for objectSyncController")
		os.Exit(1)
	}

	messageq := channelq.NewChannelMessageQueue(objectSyncController)

	// start dispatch message from the cloud to edge node
	go messageq.DispatchMessage()

	// check whether the certificates exist in the local directory,
	// and then check whether certificates exist in the secret, generate if they don't exist
	if err := httpserver.PrepareAllCerts(); err != nil {
		klog.Fatal(err)
	}
	// TODO: Will improve in the future
	DoneTLSTunnelCerts <- true
	close(DoneTLSTunnelCerts)

	// generate Token
	if err := httpserver.GenerateToken(); err != nil {
		klog.Fatal(err)
	}

	// HttpServer mainly used to issue certificates for the edge
	go httpserver.StartHTTPServer()

	servers.StartCloudHub(messageq)

	if hubconfig.Config.UnixSocket.Enable {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(hubconfig.Config.UnixSocket.Address)
	}
}

func newObjectSyncController() *hubconfig.ObjectSyncController {
	config, err := buildConfig()
	if err != nil {
		klog.Errorf("Failed to build config, err: %v", err)
		os.Exit(1)
	}

	crdClient := versioned.NewForConfigOrDie(config)
	crdFactory := crdinformerfactory.NewSharedInformerFactory(crdClient, 0)

	clusterObjectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	objectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ObjectSyncs()

	sc := &hubconfig.ObjectSyncController{
		CrdClient: crdClient,

		ClusterObjectSyncInformer: clusterObjectSyncInformer,
		ObjectSyncInformer:        objectSyncInformer,

		ClusterObjectSyncSynced: clusterObjectSyncInformer.Informer().HasSynced,
		ObjectSyncSynced:        objectSyncInformer.Informer().HasSynced,

		ClusterObjectSyncLister: clusterObjectSyncInformer.Lister(),
		ObjectSyncLister:        objectSyncInformer.Lister(),
	}

	go sc.ClusterObjectSyncInformer.Informer().Run(beehiveContext.Done())
	go sc.ObjectSyncInformer.Informer().Run(beehiveContext.Done())

	return sc
}

// build Config from flags
func buildConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(hubconfig.Config.KubeAPIConfig.Master,
		hubconfig.Config.KubeAPIConfig.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(hubconfig.Config.KubeAPIConfig.QPS)
	kubeConfig.Burst = int(hubconfig.Config.KubeAPIConfig.Burst)
	kubeConfig.ContentType = "application/json"

	return kubeConfig, nil
}
