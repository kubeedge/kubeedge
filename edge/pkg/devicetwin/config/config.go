package config

import (
	"sync"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var (
	once sync.Once
	c    Config
)

func InitDeviceTwinConfig(e *edgecoreconfig.EdgedConfig) {
	once.Do(func() {
		if e != nil {
			c.Edged = *e
		}
	})
}

type Config struct {
	Edged edgecoreconfig.EdgedConfig
}

func Conf() *Config {
	return &c
}
