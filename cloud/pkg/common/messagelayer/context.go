/*
Copyright 2022 The KubeEdge Authors.

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

package messagelayer

import (
	"strings"
	"sync"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
)

// MessageLayer define all functions that message layer must implement
type MessageLayer interface {
	Send(message model.Message) error
	Receive() (model.Message, error)
	Response(message model.Message) error
}

// ContextMessageLayer build on context
type ContextMessageLayer struct {
	// SendModuleName indicates which module will send message to
	SendModuleName string
	// SendRouterModuleName indicates which module will send router message to
	SendRouterModuleName string
	// ReceiveModuleName indicates which module will receive message from
	ReceiveModuleName string
	// ResponseModuleName indicates which module will response message to
	ResponseModuleName string
}

// Send message
func (cml *ContextMessageLayer) Send(message model.Message) error {
	module := cml.SendModuleName
	// if message is rule/ruleEndpoint type, send to router module.
	if len(cml.SendRouterModuleName) != 0 && isRouterMsg(message) {
		module = cml.SendRouterModuleName
	}
	beehiveContext.Send(module, message)
	return nil
}

// Receive message
func (cml *ContextMessageLayer) Receive() (model.Message, error) {
	return beehiveContext.Receive(cml.ReceiveModuleName)
}

// Response message
func (cml *ContextMessageLayer) Response(message model.Message) error {
	beehiveContext.Send(cml.ResponseModuleName, message)
	return nil
}

func isRouterMsg(message model.Message) bool {
	resourceArray := strings.Split(message.GetResource(), constants.ResourceSep)
	return len(resourceArray) == 2 && (resourceArray[0] == model.ResourceTypeRule || resourceArray[0] == model.ResourceTypeRuleEndpoint)
}

var (
	edgeControllerOnce         sync.Once
	edgeControllerMessageLayer MessageLayer

	deviceControllerOnce         sync.Once
	deviceControllerMessageLayer MessageLayer

	dynamicControllerOnce         sync.Once
	dynamicControllerMessageLayer MessageLayer

	nodeUpgradeJobControllerOnce         sync.Once
	nodeUpgradeJobControllerMessageLayer MessageLayer
)

func EdgeControllerMessageLayer() MessageLayer {
	edgeControllerOnce.Do(func() {
		edgeControllerMessageLayer = &ContextMessageLayer{
			SendModuleName:       modules.CloudHubModuleName,
			SendRouterModuleName: modules.RouterModuleName,
			ReceiveModuleName:    modules.EdgeControllerModuleName,
			ResponseModuleName:   modules.CloudHubModuleName,
		}
	})
	return edgeControllerMessageLayer
}

func DeviceControllerMessageLayer() MessageLayer {
	deviceControllerOnce.Do(func() {
		deviceControllerMessageLayer = &ContextMessageLayer{
			SendModuleName:     modules.CloudHubModuleName,
			ReceiveModuleName:  modules.DeviceControllerModuleName,
			ResponseModuleName: modules.CloudHubModuleName,
		}
	})
	return deviceControllerMessageLayer
}

func DynamicControllerMessageLayer() MessageLayer {
	dynamicControllerOnce.Do(func() {
		dynamicControllerMessageLayer = &ContextMessageLayer{
			SendModuleName:     modules.CloudHubModuleName,
			ReceiveModuleName:  modules.DynamicControllerModuleName,
			ResponseModuleName: modules.CloudHubModuleName,
		}
	})
	return dynamicControllerMessageLayer
}

func NodeUpgradeJobControllerMessageLayer() MessageLayer {
	nodeUpgradeJobControllerOnce.Do(func() {
		nodeUpgradeJobControllerMessageLayer = &ContextMessageLayer{
			SendModuleName:     modules.CloudHubModuleName,
			ReceiveModuleName:  modules.NodeUpgradeJobControllerModuleName,
			ResponseModuleName: modules.CloudHubModuleName,
		}
	})
	return nodeUpgradeJobControllerMessageLayer
}
