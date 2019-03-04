package config

import (
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/common/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/common/beehive/pkg/common/log"
)

// MessageLayer used, context or ssmq, default is context
var MessageLayer string

func init() {
	if ml, err := config.CONFIG.GetValue("message-layer").ToString(); err != nil {
		MessageLayer = constants.DefaultMessageLayer
	} else {
		MessageLayer = ml
	}
	log.LOGGER.Infof("message layer: %s", MessageLayer)
}
