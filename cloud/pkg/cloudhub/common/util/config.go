package util

import (
	"github.com/kubeedge/beehive/pkg/common/config"
)

// HubConfig is the config for entire CloudHub
var HubConfig *Config

func init() {
	HubConfig = &Config{}
	HubConfig.Address, _ = config.CONFIG.GetValue("cloudhub.address").ToString()
	HubConfig.Port, _ = config.CONFIG.GetValue("cloudhub.port").ToInt()
	HubConfig.KeepaliveInterval, _ = config.CONFIG.GetValue("cloudhub.keepalive-interval").ToInt()
	HubConfig.WriteTimeout, _ = config.CONFIG.GetValue("cloudhub.write-timeout").ToInt()
	HubConfig.NodeLimit, _ = config.CONFIG.GetValue("cloudhub.node-limit").ToInt()
}

// Config represents configuration options for http access
type Config struct {
	Address           string
	Port              int
	KeepaliveInterval int
	Ca                []byte
	Cert              []byte
	Key               []byte
	WriteTimeout      int
	NodeLimit         int
}
