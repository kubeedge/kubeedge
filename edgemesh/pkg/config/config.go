package config

import (
	"sync"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var (
	once sync.Once
	c    Config
)

func InitEdgeMeshConfig(m *edgecoreconfig.MeshConfig) {
	once.Do(func() {
		if m != nil {
			c.Mesh = *m
		}
	})
}

func Conf() *Config {
	return &c
}

type Config struct {
	Mesh edgecoreconfig.MeshConfig
}
