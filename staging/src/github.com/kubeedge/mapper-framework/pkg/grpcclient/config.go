package grpcclient

import (
	"github.com/kubeedge/Template/pkg/config"
)

var cfg *config.Config

func Init(c *config.Config) {
	cfg = &config.Config{}
	cfg = c
}
