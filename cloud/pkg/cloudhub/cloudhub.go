package cloudhub

import (
	"context"
	"io/ioutil"
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	chconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
)

type cloudHub struct {
	cancel context.CancelFunc
}

func Register() {
	core.Register(&cloudHub{})
}

func (a *cloudHub) Name() string {
	return "cloudhub"
}

func (a *cloudHub) Group() string {
	return "cloudhub"
}

func (a *cloudHub) Start() {
	var ctx context.Context
	ctx, a.cancel = context.WithCancel(context.Background())

	initHubConfig()

	messageq := channelq.NewChannelMessageQueue()

	// start dispatch message from the cloud to edge node
	go messageq.DispatchMessage(ctx)

	// start the cloudhub server
	if util.HubConfig.ProtocolWebsocket {
		go servers.StartCloudHub(servers.ProtocolWebsocket, messageq)
	}

	if util.HubConfig.ProtocolQuic {
		go servers.StartCloudHub(servers.ProtocolQuic, messageq)
	}

	if util.HubConfig.ProtocolUDS {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(util.HubConfig)
	}

}

func (a *cloudHub) Cleanup() {
	a.cancel()
}

func initHubConfig() {
	cafile, err := config.CONFIG.GetValue("cloudhub.ca").ToString()
	if err != nil {
		klog.Infof("missing cloudhub.ca configuration key, loading default path and filename ./%s", chconfig.DefaultCAFile)
		cafile = chconfig.DefaultCAFile
	}

	certfile, err := config.CONFIG.GetValue("cloudhub.cert").ToString()
	if err != nil {
		klog.Infof("missing cloudhub.cert configuration key, loading default path and filename ./%s", chconfig.DefaultCertFile)
		certfile = chconfig.DefaultCertFile
	}

	keyfile, err := config.CONFIG.GetValue("cloudhub.key").ToString()
	if err != nil {
		klog.Infof("missing cloudhub.key configuration key, loading default path and filename ./%s", chconfig.DefaultKeyFile)
		keyfile = chconfig.DefaultKeyFile
	}

	errs := make([]string, 0)

	util.HubConfig = &util.Config{}
	util.HubConfig.ProtocolWebsocket, _ = config.CONFIG.GetValue("cloudhub.protocol_websocket").ToBool()
	util.HubConfig.ProtocolQuic, _ = config.CONFIG.GetValue("cloudhub.protocol_quic").ToBool()
	if !util.HubConfig.ProtocolWebsocket && !util.HubConfig.ProtocolQuic {
		util.HubConfig.ProtocolWebsocket = true
	}
	util.HubConfig.ProtocolUDS, _ = config.CONFIG.GetValue("cloudhub.enable_uds").ToBool()

	util.HubConfig.Address, _ = config.CONFIG.GetValue("cloudhub.address").ToString()
	util.HubConfig.Port, _ = config.CONFIG.GetValue("cloudhub.port").ToInt()
	util.HubConfig.QuicPort, _ = config.CONFIG.GetValue("cloudhub.quic_port").ToInt()
	util.HubConfig.MaxIncomingStreams, _ = config.CONFIG.GetValue("cloudhub.max_incomingstreams").ToInt()
	util.HubConfig.UDSAddress, _ = config.CONFIG.GetValue("cloudhub.uds_address").ToString()
	util.HubConfig.KeepaliveInterval, _ = config.CONFIG.GetValue("cloudhub.keepalive-interval").ToInt()
	util.HubConfig.WriteTimeout, _ = config.CONFIG.GetValue("cloudhub.write-timeout").ToInt()
	util.HubConfig.NodeLimit, _ = config.CONFIG.GetValue("cloudhub.node-limit").ToInt()

	util.HubConfig.Ca, err = ioutil.ReadFile(cafile)
	if err != nil {
		errs = append(errs, err.Error())
	}
	util.HubConfig.Cert, err = ioutil.ReadFile(certfile)
	if err != nil {
		errs = append(errs, err.Error())
	}
	util.HubConfig.Key, err = ioutil.ReadFile(keyfile)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		klog.Errorf("cloudhub failed with errors : %v", errs)
		os.Exit(1)
	}
}
