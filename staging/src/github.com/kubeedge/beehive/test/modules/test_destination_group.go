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

package modules

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

//Constant for test module destination group name
const (
	DestinationGroupModule = "destinationgroupmodule"
)

type testModuleDestGroup struct {
}

func (m *testModuleDestGroup) Enable() bool {
	return true
}

func init() {
	core.Register(&testModuleDestGroup{})
}

func (*testModuleDestGroup) Name() string {
	return DestinationGroupModule
}

func (*testModuleDestGroup) Group() string {
	return DestinationGroup
}

func (m *testModuleDestGroup) Start() {
	message, err := beehiveContext.Receive(DestinationGroupModule)
	fmt.Printf("destination group module receive message:%v error:%v\n", message, err)
	if message.IsSync() {
		resp := message.NewRespByMessage(&message, "10 years old")
		beehiveContext.SendResp(*resp)
	}
}
