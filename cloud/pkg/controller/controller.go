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
package controller

import (
	"os"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/controller"
)

// Controller use beehive context message layer
type Controller struct{}

func init() {
	edgeController := Controller{}
	core.Register(&edgeController)
}

// Name of controller
func (ctl *Controller) Name() string {
	return constants.EdgeControllerModuleName
}

// Group of controller
func (ctl *Controller) Group() string {
	return constants.EdgeControllerModuleName
}

// Start controller
func (ctl *Controller) Start(c *bcontext.Context) {
	config.Context = c
	upstream, err := controller.NewUpstreamController()
	if err != nil {
		log.LOGGER.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()

	for {
		stopChannel := make(<-chan struct{})
		downstream, err := controller.NewDownstreamController()
		if err != nil {
			log.LOGGER.Warnf("new downstream controller failed with error: %s", err)
			continue
		}
		downstream.Start()

		<-stopChannel
		log.LOGGER.Warnf("elected slave, stopping downstream controller...")
		downstream.Stop()
		log.LOGGER.Warnf("downstream controller stopped")
	}
}

// Cleanup controller
func (ctl *Controller) Cleanup() {
	// TODO
}
