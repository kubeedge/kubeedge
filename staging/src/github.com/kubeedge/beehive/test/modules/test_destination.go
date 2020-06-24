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

//Constants for module name and group
const (
	DestinationModule = "destinationmodule"
	DestinationGroup  = "destinationgroup"
)

type testModuleDest struct {
}

func (m *testModuleDest) Enable() bool {
	return true
}

func init() {
	core.Register(&testModuleDest{})
}

func (*testModuleDest) Name() string {
	return DestinationModule
}

func (*testModuleDest) Group() string {
	return DestinationGroup
}

func (m *testModuleDest) Start() {
	message, err := beehiveContext.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	message, err = beehiveContext.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	resp := message.NewRespByMessage(&message, "fine")
	if message.IsSync() {
		beehiveContext.SendResp(*resp)
	}

	message, err = beehiveContext.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	if message.IsSync() {
		resp = message.NewRespByMessage(&message, "fine")
		beehiveContext.SendResp(*resp)
	}

	//message, err = c.Receive(DestinationModule)
	//fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	//if message.IsSync() {
	//	resp = message.NewRespByMessage(&message, "20 years old")
	//	c.SendResp(*resp)
	//}
}
