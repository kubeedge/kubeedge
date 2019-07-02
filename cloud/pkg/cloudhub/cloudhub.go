/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package cloudhub
package cloudhub

import (
	"io/ioutil"
	"os"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	chconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
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

	initHubConfig()

	eventq := channelq.NewChannelEventQueue(c)

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

func initHubConfig() {
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
}
