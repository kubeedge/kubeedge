package cloudhub

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"io/ioutil"
	"os"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	chconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
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

	cafile, err := config.CONFIG.GetValue("cloudhub.ca").ToString()
	if err != nil {
		log.LOGGER.Info("missing cloudhub.ca configuration key, loading default path and filename ./" + chconfig.DefaultCAFile)
		cafile = chconfig.DefaultCAFile
	}

	certfile, err := config.CONFIG.GetValue("cloudhub.cert").ToString()
	if err != nil {
		log.LOGGER.Info("missing cloudhub.cert configuration key, loading default path and filename ./" + chconfig.DefaultCertFile)
		certfile = chconfig.DefaultCertFile
	}

	keyfile, err := config.CONFIG.GetValue("cloudhub.key").ToString()
	if err != nil {
		log.LOGGER.Info("missing cloudhub.key configuration key, loading default path and filename ./" + chconfig.DefaultKeyFile)
		keyfile = chconfig.DefaultKeyFile
	}

	errs := make([]string, 0)

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
		log.LOGGER.Errorf("cloudhub failed with errors : %v", errs)
		os.Exit(1)
	}

	eventq, err := channelq.NewChannelEventQueue(c)
	// start the cloudhub server
	if util.HubConfig.ProtocolWebsocket {
		go servers.StartCloudHub(servers.ProtocolWebsocket, eventq, c)
	}

	if util.HubConfig.ProtocolQuic {
		go servers.StartCloudHub(servers.ProtocolQuic, eventq, c)
	}

	stopchan := make(chan bool)
	<-stopchan
}

func (a *cloudHub) Cleanup() {
	a.context.Cleanup(a.Name())
}
