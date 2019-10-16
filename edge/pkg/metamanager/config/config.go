package config

import (
	"sync"

	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var (
	once sync.Once
	c    Config
)

func InitMetamanagerConfig(m *edgecoreconfig.Metamanager) {
	once.Do(func() {
		if m != nil {
			c.Meta = *m
			c.Connected = c.Meta.EdgeSite
		}
	})
}

type Config struct {
	Meta      edgecoreconfig.Metamanager
	Connected bool
}

func Conf() *Config {
	return &c
}
