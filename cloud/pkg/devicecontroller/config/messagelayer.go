package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// MessageLayer used, context or ssmq, default is context
var MessageLayer string

func InitMessageLayerConfig() {
	if ml, err := config.CONFIG.GetValue("devicecontroller.message-layer").ToString(); err != nil {
		MessageLayer = constants.DefaultMessageLayer
	} else {
		MessageLayer = ml
	}
	klog.Infof("Message layer: %s", MessageLayer)
}
