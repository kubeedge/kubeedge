package config

import (
	"github.com/kubeedge/kubeedge/common/constants"
)

// MessageLayer used, context or ssmq, default is context
var MessageLayer string

func InitMessageLayerConfig() {
	MessageLayer = constants.DefaultMessageLayer
}
