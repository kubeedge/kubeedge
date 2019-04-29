package cloudhub

import (
	"io/ioutil"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	chconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/wsserver"
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
		cafile = chconfig.DefaultCAFile
	}

	certfile, err := config.CONFIG.GetValue("cloudhub.cert").ToString()
	if err != nil {
		certfile = chconfig.DefaultCertFile
	}

	keyfile, err := config.CONFIG.GetValue("cloudhub.key").ToString()
	if err != nil {
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

	if len(errs) <= 0 {
		//  TBD : add code for to sync graceful exit of modules
		return
	}

	eventq, err := channelq.NewChannelEventQueue(c)
	// start the cloudhub server
	wsserver.StartCloudHub(util.HubConfig, eventq)
	wsserver.EventHandler.Context = c
	stopchan := make(chan bool)
	<-stopchan
}

func (a *cloudHub) Cleanup() {
	a.context.Cleanup(a.Name())
}
