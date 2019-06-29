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
package devicetwin

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
)

//DeviceTwin the module
type DeviceTwin struct {
	context      *context.Context
	dtcontroller *DTController
}

func init() {
	dtclient.InitDBTable()
	dt := DeviceTwin{}
	core.Register(&dt)
}

//Name get name of the module
func (dt *DeviceTwin) Name() string {
	return "twin"
}

//Group get group of the module
func (dt *DeviceTwin) Group() string {
	return modules.TwinGroup
}

//Start run the module
func (dt *DeviceTwin) Start(c *context.Context) {
	controller, err := InitDTController(c)
	if err != nil {
		log.LOGGER.Errorf("Start device twin failed, due to %v", err)
	}
	dt.dtcontroller = controller
	dt.context = c
	err = controller.Start()
	if err != nil {
		log.LOGGER.Errorf("Start device twin failed, due to %v", err)
	}
}

//Cleanup clean resource after quit
func (dt *DeviceTwin) Cleanup() {
	dt.dtcontroller.Stop <- true
	dt.context.Cleanup(dt.Name())
}
