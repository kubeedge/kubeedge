package config

import "sync"

var (
	once sync.Once
	c    Config
)

func InitEventbusConfig() {
	once.Do(func() {
	})
}

func Conf() *Config {
	return &c
}

type Config struct {
}
