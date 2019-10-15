package config

import (
	"sync"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var (
	once sync.Once
	c    Config
)

func InitEdgedConfig(e *edgecoreconfig.EdgedConfig) {
	once.Do(func() {
		if e != nil {
			c.EdgedConfig = *e
		}
	})
}

type Config struct {
	edgecoreconfig.EdgedConfig
}

func Conf() *Config {
	return &c
}
