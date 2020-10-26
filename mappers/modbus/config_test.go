package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	file := "./config.yaml"

	config := Config{}
	config.Parse(file)

	assert.Equal(t, "tcp://127.0.0.1:1883", config.Mqtt.ServerAddress)
	assert.Equal(t, "/opt/kubeedge/deviceProfile.json", config.Configmap)
}
