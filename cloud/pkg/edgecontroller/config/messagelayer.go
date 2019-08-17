package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

// MessageLayer used, context or ssmq, default is context
var MessageLayer string

func InitMessageLayerConfig() {
	if ml, err := config.CONFIG.GetValue("controller.message-layer").ToString(); err != nil {
		MessageLayer = constants.DefaultMessageLayer
	} else {
		MessageLayer = ml
	}
	klog.Infof("message layer: %s", MessageLayer)
}
