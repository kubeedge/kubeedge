package cloudhub

import (
	"os"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/dispatcher"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var DoneTLSTunnelCerts = make(chan bool, 1)

type cloudHub struct {
	enable               bool
	informersSyncedFuncs []cache.InformerSynced

	messageHandler handler.Handler
	dispatcher     dispatcher.MessageDispatcher
}

var _ core.Module = (*cloudHub)(nil)

func newCloudHub(enable bool) *cloudHub {
	crdFactory := informers.GetInformersManager().GetKubeEdgeInformerFactory()
	// declare used informer
	clusterObjectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ClusterObjectSyncs()
	objectSyncInformer := crdFactory.Reliablesyncs().V1alpha1().ObjectSyncs()

	sessionManager := session.NewSessionManager(hubconfig.Config.NodeLimit)

	messageDispatcher := dispatcher.NewMessageDispatcher(
		sessionManager, objectSyncInformer.Lister(),
		clusterObjectSyncInformer.Lister(), client.GetCRDClient())

	messageHandler := handler.NewMessageHandler(
		int(hubconfig.Config.KeepaliveInterval),
		sessionManager, client.GetCRDClient(), messageDispatcher)

	ch := &cloudHub{
		enable:         enable,
		dispatcher:     messageDispatcher,
		messageHandler: messageHandler,
	}

	ch.informersSyncedFuncs = append(ch.informersSyncedFuncs, clusterObjectSyncInformer.Informer().HasSynced)
	ch.informersSyncedFuncs = append(ch.informersSyncedFuncs, objectSyncInformer.Informer().HasSynced)

	return ch
}

func Register(hub *v1alpha1.CloudHub) {
	hubconfig.InitConfigure(hub)
	core.Register(newCloudHub(hub.Enable))
}

func (ch *cloudHub) Name() string {
	return modules.CloudHubModuleName
}

func (ch *cloudHub) Group() string {
	return modules.CloudHubModuleGroup
}

// Enable indicates whether enable this module
func (ch *cloudHub) Enable() bool {
	return ch.enable
}

func (ch *cloudHub) Start() {
	if !cache.WaitForCacheSync(beehiveContext.Done(), ch.informersSyncedFuncs...) {
		klog.Errorf("unable to sync caches for objectSyncController")
		os.Exit(1)
	}

	// start dispatch message from the cloud to edge node
	go ch.dispatcher.DispatchDownstream()

	// check whether the certificates exist in the local directory,
	// and then check whether certificates exist in the secret, generate if they don't exist
	if err := httpserver.PrepareAllCerts(); err != nil {
		klog.Exit(err)
	}
	// TODO: Will improve in the future
	DoneTLSTunnelCerts <- true
	close(DoneTLSTunnelCerts)

	// generate Token
	if err := httpserver.GenerateToken(); err != nil {
		klog.Exit(err)
	}

	// HttpServer mainly used to issue certificates for the edge
	go httpserver.StartHTTPServer()

	servers.StartCloudHub(ch.messageHandler)

	if hubconfig.Config.UnixSocket.Enable {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(hubconfig.Config.UnixSocket.Address)
	}
}
