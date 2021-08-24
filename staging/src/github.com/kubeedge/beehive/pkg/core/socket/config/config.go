package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// SocketConfig socket config
type SocketConfig struct {
	ModuleName    string `yaml:"module"`
	Server        bool   `yaml:"server"`
	Address       string `yaml:"address"`
	SocketType    string `yaml:"sockettype,omitempty"`
	ConnNumberMax int    `yaml:"connmax"`
	BufferSize    int    `yaml:"buffersize,omitempty"`
	CaRoot        string `yaml:"ca,omitempty"`
	Cert          string `yaml:"cert,omitempty"`
	Key           string `yaml:"key,omitempty"`
}

// BuildinModuleConfig buildin module config
type BuildinModuleConfig struct {
	// socket
	socketList []SocketConfig
}

func init() {
	var filepath string
	filepath = os.Getenv("SOCKET_MODULE_CONFIG")
	if filepath == "" {
		filepath = "/etc/kubeedge/config/socket_module.yaml"
		if _, err := os.Stat(filepath); err != nil {
			return
		}
	}

	InitBuildinModuleConfig(filepath)
}

var (
	buildinModuleConfig *BuildinModuleConfig
)

// InitBuildinModuleConfig init buildin module config
func InitBuildinModuleConfig(filepath string) *BuildinModuleConfig {
	moduleConfig := BuildinModuleConfig{}
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		klog.Errorf("failed to read file %v: %v", filepath, err)
		return nil
	}
	err = yaml.Unmarshal(data, &moduleConfig.socketList)
	if err != nil {
		klog.Errorf("failed to yaml unmarshal config: %v", err)
		return nil
	}

	return &moduleConfig
}

// GetClientSocketConfig get client socket config
func GetClientSocketConfig(module string) (SocketConfig, error) {
	for _, socketConfig := range buildinModuleConfig.socketList {
		if socketConfig.ModuleName == module && !socketConfig.Server {
			return socketConfig, nil
		}
	}

	return SocketConfig{}, fmt.Errorf("failed to get socket config by name(%s)", module)
}

// GetServerSocketConfig get server socket config
func GetServerSocketConfig() ([]SocketConfig, error) {
	var serversSocket []SocketConfig
	for _, socketConfig := range buildinModuleConfig.socketList {
		if socketConfig.Server {
			serversSocket = append(serversSocket, socketConfig)
		}
	}

	if len(serversSocket) != 0 {
		return serversSocket, nil
	}
	return []SocketConfig{}, fmt.Errorf("failed to get socket config")
}
