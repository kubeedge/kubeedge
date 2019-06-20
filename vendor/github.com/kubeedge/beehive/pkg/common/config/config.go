package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ServiceComb/go-archaius"
	"github.com/ServiceComb/go-archaius/sources/file-source"
	"github.com/go-chassis/go-archaius/core"

	"github.com/kubeedge/beehive/pkg/common/util"
)

//constants to define config paths
const (
	ParameterConfigPath     = "config-path"
	EnvironmentalConfigPath = "GOARCHAIUS_CONFIG_PATH"
)

// CONFIG conf
var CONFIG goarchaius.ConfigurationFactory

func init() {
	// create go-archaius object
	configFactory, err := goarchaius.NewConfigFactory(nil)
	if err != nil {
		fmt.Printf("failed to new config factory,  error: %+v", err)
	}
	CONFIG = configFactory
	// init go-archaius
	err = CONFIG.Init()
	if err != nil {
		fmt.Printf("failed to init config factory,  error: %+v", err)
	}
	//Add yaml files as config source
	fSource := filesource.NewYamlConfigurationSource()
	confLocation := GetConfigDirectory() + "/conf"
	err = filepath.Walk(confLocation, func(location string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(location))
		if ext == ".yml" || ext == ".yaml" {
			fSource.AddFileSource(location, 0)
			fmt.Printf("New file source added for configuration: %s", location)
		}
		return nil
	})
	CONFIG.AddSource(fSource)
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v", err)
	}
	CONFIG.GetConfigurations()

}

// GetConfigDirectory gets the configuration file path
func GetConfigDirectory() string {
	if config, err := CONFIG.GetValue(ParameterConfigPath).ToString(); err == nil {
		return config
	}

	if config, err := CONFIG.GetValue(EnvironmentalConfigPath).ToString(); err == nil {
		return config
	}

	return util.GetCurrentDirectory()
}

//ChangeCallback is interface to change callback of config
type ChangeCallback interface {
	Callback(k string, v interface{})
}

//ConfigChangeCallbacks is array of changecallbacks
var ConfigChangeCallbacks []ChangeCallback

//AddConfigChangeCallback adds a config change callback
func AddConfigChangeCallback(cb ChangeCallback) {
	ConfigChangeCallbacks = append(ConfigChangeCallbacks, cb)
}

//EventListener is object to define eventlistener
type EventListener struct {
	Name string
}

//Event is a method get config value and logs it
func (e EventListener) Event(event *core.Event) {
	configValue := CONFIG.GetConfigurationByKey(event.Key)
	for _, c := range ConfigChangeCallbacks {
		c.Callback(event.Key, configValue)
		fmt.Printf("config value %v | %v", event.Key, configValue)
	}
}
