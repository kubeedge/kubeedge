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
	ModuleName    string `json:"module"`
	Server        bool   `json:"server"`
	Address       string `json:"address"`
	SocketType    string `json:"sockettype,omitempty"`
	ConnNumberMax int    `json:"connmax"`
	BufferSize    int    `json:"buffersize,omitempty"`
	CaRoot        string `json:"ca,omitempty"`
	Cert          string `json:"cert,omitempty"`
	Key           string `json:"key,omitempty"`
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

	buildinModuleConfig = InitBuildinModuleConfig(filepath)
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
