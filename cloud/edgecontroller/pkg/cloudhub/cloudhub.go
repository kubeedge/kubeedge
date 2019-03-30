package cloudhub

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/servers"
)

type cloudHub struct {
	context *context.Context
}

func init() {
	core.Register(&cloudHub{})
}

func (a *cloudHub) Name() string {
	return "cloudhub"
}

func (a *cloudHub) Group() string {
	return "cloudhub"
}

func (a *cloudHub) Start(c *context.Context) {
	a.context = c
	var err error
	caI := config.CONFIG.GetConfigurationByKey("cloudhub.ca")
	certI := config.CONFIG.GetConfigurationByKey("cloudhub.cert")
	keyI := config.CONFIG.GetConfigurationByKey("cloudhub.key")

	util.HubConfig.Ca, err = ioutil.ReadFile(caI.(string))
	if err != nil {
		panic(err)
	}

	util.HubConfig.Cert, err = ioutil.ReadFile(certI.(string))
	if err != nil {
		panic(err)
	}
	util.HubConfig.Key, err = ioutil.ReadFile(keyI.(string))
	if err != nil {
		panic(err)
	}

	// init filter
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(util.HubConfig.Ca)
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}

	eventq, err := channelq.NewChannelEventQueue(c)

	// start the cloudhub server
	if util.HubConfig.ProtocolWebsocket {
		go servers.StartCloudHub(servers.PROTOCOL_WEBSOCKET, eventq, c)
	}

	if util.HubConfig.ProtocolQuic {
		go servers.StartCloudHub(servers.PROTOCOL_QUIC, eventq, c)
	}

	stopchan := make(chan bool)
	<-stopchan
}

func (a *cloudHub) Cleanup() {
	a.context.Cleanup(a.Name())
}
