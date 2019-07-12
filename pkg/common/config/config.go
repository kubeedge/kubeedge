package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/common/util"

	archaius "github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/core"
	commandlinesource "github.com/go-chassis/go-archaius/sources/commandline-source"
	envconfigsource "github.com/go-chassis/go-archaius/sources/enviromentvariable-source"
	memoryconfigsource "github.com/go-chassis/go-archaius/sources/memory-source"
	yaml "gopkg.in/yaml.v2"
)

//constants to define config paths
const (
	ParameterConfigPath     = "config-path"
	EnvironmentalConfigPath = "GOARCHAIUS_CONFIG_PATH"
)

// CONFIG conf
var CONFIG archaius.ConfigurationFactory
var once = sync.Once{}

// config file  only support .yml or .yaml  !
func InitializeConfig() {
	once.Do(func() {
		err := archaius.Init()
		if err != nil {
			log.LOGGER.Errorf("archaius init failed!")
		}
		CONFIG = archaius.GetConfigFactory()
		ms := memoryconfigsource.NewMemoryConfigurationSource()
		CONFIG.AddSource(ms)

		cmdSource := commandlinesource.NewCommandlineConfigSource()
		CONFIG.AddSource(cmdSource)

		envSource := envconfigsource.NewEnvConfigurationSource()
		CONFIG.AddSource(envSource)
		confLocation := getConfigDirectory() + "/conf"
		_, err = os.Stat(confLocation)
		if !os.IsExist(err) {
			os.Mkdir(confLocation, os.ModePerm)
		}
		err = filepath.Walk(confLocation, func(location string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}
			ext := strings.ToLower(path.Ext(location))
			if ext == ".yml" || ext == ".yaml" {
				archaius.AddFile(location)
			}
			return nil
		})

		if err != nil {
			log.LOGGER.Errorf("filepath.Walk() returned %s\n", err.Error())
		}
	})
}

func init() {
	if log.LOGGER == nil {
		log.InitializeLogger()
	}
	InitializeConfig()
}

// get the configuration file path
func getConfigDirectory() string {
	if config, err := CONFIG.GetValue(ParameterConfigPath).ToString(); err == nil {
		return config
	}

	if config, err := CONFIG.GetValue(EnvironmentalConfigPath).ToString(); err == nil {
		return config
	}

	return util.GetCurrentDirectory()
}

func StringVar(p *string, name string, value string, usage string) {
	if str, err := CONFIG.GetValue(name).ToString(); err == nil {
		*p = str
	} else {
		*p = value
	}
}

func StringSliceVar(slice *[]string, name string, value []string, usage string) {
	if strs, err := CONFIG.GetValue(name).ToStringSlice(); err == nil {
		for _, v := range strs {
			*slice = append(*slice, v)
		}
	}
}

func InterfaceSliceVar(slice interface{}, name string, value interface{}, usage string) {
	if sliceTemp, err := CONFIG.GetValue(name).ToSlice(); err == nil {
		if sliceBytes, err := yaml.Marshal(sliceTemp); err == nil {
			err = yaml.Unmarshal(sliceBytes, slice)
			if err != nil {
				fmt.Printf("failed to marshal, error:%+v", err)
			}
		}
	}
}

func IntVar(p *int, name string, value int, usage string) {
	if v, err := CONFIG.GetValue(name).ToInt(); err == nil {
		*p = v
	} else {
		*p = value
	}
}

func BoolVar(p *bool, name string, value bool, usage string) {
	if v, err := CONFIG.GetValue(name).ToBool(); err == nil {
		*p = v
	} else {
		*p = value
	}
}

func GetBool(key string, defaultValue bool) bool {
	if CONFIG == nil {
		InitializeConfig()
	}
	return archaius.GetBool(key, defaultValue)
}

func GetString(key string, defaultValue string) string {
	if CONFIG == nil {
		InitializeConfig()
	}
	return archaius.GetString(key, defaultValue)
}

func GetInt(key string, defaultValue int) int {
	if CONFIG == nil {
		InitializeConfig()
	}
	return archaius.GetInt(key, defaultValue)
}

func Get(key string) interface{} {
	if CONFIG == nil {
		InitializeConfig()
	}
	return archaius.Get(key)
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
