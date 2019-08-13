package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core/context"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// ContextSendModule is the name send message to
var ContextSendModule string

// ContextReceiveModule is the name receive message from
var ContextReceiveModule string

// ContextResponseModule is the name response message from
var ContextResponseModule string

// Context is beehive context used to send message
var Context *context.Context

func InitContextConfig() {
	if smn, err := config.CONFIG.GetValue("devicecontroller.context.send-module").ToString(); err != nil {
		ContextSendModule = constants.DefaultContextSendModuleName
	} else {
		ContextSendModule = smn
	}
	klog.Infof("Send module name: %s", ContextSendModule)

	if rmn, err := config.CONFIG.GetValue("devicecontroller.context.receive-module").ToString(); err != nil {
		ContextReceiveModule = constants.DefaultContextReceiveModuleName
	} else {
		ContextReceiveModule = rmn
	}
	klog.Infof("Receive module name: %s", ContextReceiveModule)

	if rmn, err := config.CONFIG.GetValue("devicecontroller.context.response-module").ToString(); err != nil {
		ContextResponseModule = constants.DefaultContextResponseModuleName
	} else {
		ContextResponseModule = rmn
	}
	klog.Infof("Response module name: %s", ContextResponseModule)
}
