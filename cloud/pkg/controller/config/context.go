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

// Package config
package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/common/constants"
)

// ContextSendModule is the name send message to
var ContextSendModule string

// ContextReceiveModule is the name receive message from
var ContextReceiveModule string

// ContextResponseModule is the name response message from
var ContextResponseModule string

// Context ...
var Context *context.Context

func init() {
	if smn, err := config.CONFIG.GetValue("controller.context.send-module").ToString(); err != nil {
		ContextSendModule = constants.DefaultContextSendModuleName
	} else {
		ContextSendModule = smn
	}
	log.LOGGER.Infof(" send module name: %s", ContextSendModule)

	if rmn, err := config.CONFIG.GetValue("controller.context.receive-module").ToString(); err != nil {
		ContextReceiveModule = constants.DefaultContextReceiveModuleName
	} else {
		ContextReceiveModule = rmn
	}
	log.LOGGER.Infof("receive module name: %s", ContextReceiveModule)

	if rmn, err := config.CONFIG.GetValue("controller.context.response-module").ToString(); err != nil {
		ContextResponseModule = constants.DefaultContextResponseModuleName
	} else {
		ContextResponseModule = rmn
	}
	log.LOGGER.Infof("response module name: %s", ContextResponseModule)
}
