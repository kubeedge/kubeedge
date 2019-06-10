package config_test

import (
	"github.com/go-chassis/go-chassis-config"
	"github.com/stretchr/testify/assert"
	"testing"

	_ "github.com/go-chassis/go-chassis-config/configcenter"
)

func TestEnable(t *testing.T) {
	c, err := config.NewClient("config_center", config.Options{
		ServerURI: "http://127.0.0.1:30100",
	})
	assert.NoError(t, err)
	_, err = c.PullConfigs("service", "app", "1.0", "")
	assert.Error(t, err)
}
