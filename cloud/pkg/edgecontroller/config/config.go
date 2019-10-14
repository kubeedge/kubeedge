package config

import (
	"sync"

	"github.com/kubeedge/beehive/pkg/core/context"
	cconfig "github.com/kubeedge/kubeedge/cloud/pkg/apis/cloudcore/config"
)

// Context ...
var Context *context.Context

var config Config
var once sync.Once

func InitEdgeControllerConfig(e cconfig.EdgeControllerConfig) {
	once.Do(func() {
		config.EdgeControllerConfig = e
	})
}

type Config struct {
	cconfig.EdgeControllerConfig
}

func Conf() *Config {
	return &config
}
