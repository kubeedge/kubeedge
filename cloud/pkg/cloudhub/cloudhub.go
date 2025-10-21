package cloudhub

import (
	"errors"
	"fmt"
	"time"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/authorization"
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
)

var DoneTLSTunnelCerts = make(chan bool, 1)
var sessionMgr *session.Manager

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

	config := getAuthConfig()
	authorizer, err := config.New()
	if err != nil {
		panic(fmt.Sprintf("unable to create new authorizer for CloudHub: %v", err))
	}

	messageHandler := handler.NewMessageHandler(
		int(hubconfig.Config.KeepaliveInterval),
		sessionManager, client.GetCRDClient(),
		messageDispatcher, authorizer)
	sessionMgr = sessionManager

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
    klog.Info("Registering CloudHub module")
    hubconfig.InitConfigure(hub)
    
    if hubconfig.Config.UnixSocket.Enable {
        klog.Infof("UDS server enabled at: %s", hubconfig.Config.UnixSocket.Address)
    }
    klog.Infof("CloudHub node limit: %d", hubconfig.Config.NodeLimit)
    klog.Infof("CloudHub keepalive interval: %d", hubconfig.Config.KeepaliveInterval)
    
    core.Register(newCloudHub(hub.Enable))
    klog.Info("CloudHub module registered successfully")
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
    klog.Info("Starting CloudHub module")
    startTime := time.Now()
    
    if !cache.WaitForCacheSync(beehiveContext.Done(), ch.informersSyncedFuncs...) {
        klog.Exitf("unable to sync caches for objectSyncController")
    }
    klog.Infof("Cache sync completed in %v", time.Since(startTime))

    ctx := beehiveContext.GetContext()

    // start dispatch message from the cloud to edge node
    klog.Info("Starting downstream message dispatcher")
    go ch.dispatcher.DispatchDownstream()

    // check whether the certificates exist in the local directory,
    // and then check whether certificates exist in the secret, generate if they don't exist
    klog.Info("Preparing TLS certificates")
    certStart := time.Now()
    if err := httpserver.PrepareAllCerts(ctx); err != nil {
        klog.Exit(err)
    }
    klog.Infof("Certificate preparation completed in %v", time.Since(certStart))
    
    // TODO: Will improve in the future
    DoneTLSTunnelCerts <- true
    close(DoneTLSTunnelCerts)

    // generate Token
    klog.Info("Generating and refreshing authentication token")
    tokenStart := time.Now()
    if err := httpserver.GenerateAndRefreshToken(ctx); err != nil {
        klog.Exit(err)
    }
    klog.Infof("Token generation completed in %v", time.Since(tokenStart))

    // HttpServer mainly used to issue certificates for the edge
    klog.Info("Starting HTTP server for certificate issuance")
    go func() {
        if err := httpserver.StartHTTPServer(); err != nil {
            klog.Exit(err)
        }
    }()

    klog.Info("Starting CloudHub message servers")
    serverStart := time.Now()
    servers.StartCloudHub(ch.messageHandler)
    klog.Infof("CloudHub servers started in %v", time.Since(serverStart))

    if hubconfig.Config.UnixSocket.Enable {
        klog.Infof("Starting UDS server at %s", hubconfig.Config.UnixSocket.Address)
        // The uds server is only used to communicate with csi driver from kubeedge on cloud.
        // It is not used to communicate between cloud and edge.
        go udsserver.StartServer(hubconfig.Config.UnixSocket.Address)
    }
    
    totalTime := time.Since(startTime)
    klog.Infof("CloudHub module started successfully in %v", totalTime)
}


func getAuthConfig() authorization.Config {
	enabled := hubconfig.Config.Authorization != nil && hubconfig.Config.Authorization.Enable
	debug := enabled && hubconfig.Config.Authorization.Debug
	builtinInformerFactory := informers.GetInformersManager().GetKubeInformerFactory()

	var authorizationModes []string
	if enabled {
		for _, modeConfig := range hubconfig.Config.Authorization.Modes {
			switch {
			case modeConfig.Node != nil && modeConfig.Node.Enable:
				{
					authorizationModes = append(authorizationModes, modes.ModeNode)
				}
			}
		}
	}
	if len(authorizationModes) == 0 {
		authorizationModes = []string{modes.ModeAlwaysAllow}
	}

	return authorization.Config{
		Enabled:                  enabled,
		Debug:                    debug,
		AuthorizationModes:       authorizationModes,
		VersionedInformerFactory: builtinInformerFactory,
	}
}

func GetSessionManager() (*session.Manager, error) {
	if sessionMgr != nil {
		return sessionMgr, nil
	}
	return nil, errors.New("cloudhub not initialized")
}
