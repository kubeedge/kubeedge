package main

import (
	"io/ioutil"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Mqtt      Mqtt   `yaml:"mqtt,omitempty"`
	Configmap string `yaml:"configmap"`
}

type Mqtt struct {
	ServerAddress string `yaml:"server,omitempty"`
	Username      string `yaml:"username,omitempty"`
	Password      string `yaml:"password,omitempty"`
	Cert          string `yaml:"certification,omitempty"`
}

func NewDefaultConfig() *Config {
	return &Config{
		Mqtt: Mqtt{
			ServerAddress: "tcp://127.0.0.1:1883",
			Username:      "",
			Password:      "",
			Cert:          "",
		},
		Configmap: "/opt/kubeedge/deviceProfile.json",
	}
}

func (c *Config) MustParse(configFile string) {
	err := c.Parse(configFile)
	if err != nil {
		panic(err)
	}
}

func (c *Config) Parse(configFile string) error {
	cf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(cf, c)
	if err != nil {
		return err
	}
	c.parseFlags()
	return nil
}

func (c *Config) parseFlags() {
	pflag.StringVar(&c.Mqtt.ServerAddress, "mqtt-address", c.Mqtt.ServerAddress, "MQTT broker address")
	pflag.StringVar(&c.Mqtt.Username, "mqtt-username", c.Mqtt.Username, "username")
	pflag.StringVar(&c.Mqtt.Password, "mqtt-password", c.Mqtt.Password, "password")
	pflag.StringVar(&c.Mqtt.Cert, "mqtt-certification", c.Mqtt.Cert, "certification")
	pflag.Parse()
}
