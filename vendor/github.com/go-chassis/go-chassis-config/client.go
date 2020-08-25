package config

import (
	"errors"
	"fmt"

	"github.com/go-mesh/openlogging"
)

var configClientPlugins = make(map[string]func(options Options) (Client, error))

//DefaultClient is config server's client
var DefaultClient Client

//InstallConfigClientPlugin install a config client plugin
func InstallConfigClientPlugin(name string, f func(options Options) (Client, error)) {
	configClientPlugins[name] = f
	openlogging.GetLogger().Infof("Installed %s Plugin", name)
}

//Client is the interface of config server client, it has basic func to interact with config server
type Client interface {
	//PullConfigs pull all configs from remote
	PullConfigs(serviceName, version, app, env string) (map[string]interface{}, error)
	//PullConfig pull one config from remote
	PullConfig(serviceName, version, app, env, key, contentType string) (interface{}, error)
	//PullConfigsByDI pulls the configurations with customized DimensionInfo/Project
	PullConfigsByDI(dimensionInfo string) (map[string]map[string]interface{}, error)
	// PushConfigs push config to cc
	PushConfigs(data map[string]interface{}, serviceName, version, app, env string) (map[string]interface{}, error)
	// DeleteConfigsByKeys delete config for cc by keys
	DeleteConfigsByKeys(keys []string, serviceName, version, app, env string) (map[string]interface{}, error)
	//Watch get kv change results, you can compare them with local kv cache and refresh local cache
	Watch(f func(map[string]interface{}), errHandler func(err error)) error

	Options() Options
}

//NewClient create config client implementation
func NewClient(name string, options Options) (Client, error) {
	plugins := configClientPlugins[name]
	if plugins == nil {
		return nil, errors.New(fmt.Sprintf("plugin [%s] not found", name))
	}
	DefaultClient, err := plugins(options)
	if err != nil {
		return nil, err
	}
	openlogging.GetLogger().Infof("%s plugin is enabled", name)
	return DefaultClient, nil
}
