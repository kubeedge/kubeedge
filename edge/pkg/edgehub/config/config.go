package config

import (
	"fmt"
	"path"
	"sync"

	"k8s.io/klog"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var (
	once sync.Once
	c    Config
)

func InitEdgehubConfig(h *edgecoreconfig.EdgeHubConfig, e *edgecoreconfig.EdgedConfig) {
	once.Do(func() {
		if h != nil {
			c.EdgeHubConfig = *h
		}
		if e != nil {
			c.EdgedConfig = *e
		}
		c.WebSocketURL = AssemblyWebSocketURL(c.EdgeHubConfig.WebSocket.Server, c.EdgeHubConfig.Controller.ProjectId, c.EdgedConfig.HostnameOverride)
	})
}

type Config struct {
	EdgeHubConfig edgecoreconfig.EdgeHubConfig
	EdgedConfig   edgecoreconfig.EdgedConfig
	WebSocketURL  string
}

func Conf() *Config {
	return &c
}

func AssemblyWebSocketURL(server, projectid, nodename string) string {
	if projectid == "" || nodename == "" {
		klog.Errorf("Nedd project id or node name")
	}
	return fmt.Sprintf("wss://%s", path.Join(server, projectid, nodename, "events"))
}
