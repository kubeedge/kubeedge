package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// MessageLayer used, context or ssmq, default is context
var MessageLayer string

func init() {
	if ml, err := config.CONFIG.GetValue("devicecontroller.message-layer").ToString(); err != nil {
		MessageLayer = constants.DefaultMessageLayer
	} else {
		MessageLayer = ml
	}
	log.LOGGER.Infof("Message layer: %s", MessageLayer)
}
