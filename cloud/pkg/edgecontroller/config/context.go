package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
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

func InitContextConfig() {
	if smn, err := config.CONFIG.GetValue("controller.context.send-module").ToString(); err != nil {
		ContextSendModule = constants.DefaultContextSendModuleName
	} else {
		ContextSendModule = smn
	}
	klog.Infof(" send module name: %s", ContextSendModule)

	if rmn, err := config.CONFIG.GetValue("controller.context.receive-module").ToString(); err != nil {
		ContextReceiveModule = constants.DefaultContextReceiveModuleName
	} else {
		ContextReceiveModule = rmn
	}
	klog.Infof("receive module name: %s", ContextReceiveModule)

	if rmn, err := config.CONFIG.GetValue("controller.context.response-module").ToString(); err != nil {
		ContextResponseModule = constants.DefaultContextResponseModuleName
	} else {
		ContextResponseModule = rmn
	}
	klog.Infof("response module name: %s", ContextResponseModule)
}
