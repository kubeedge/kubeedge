// Package archaius provides you APIs which helps to manage files,
// remote config center configurations
package archaius

import (
	"errors"
	"os"
	"strings"

	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/sources/commandline-source"
	"github.com/go-chassis/go-archaius/sources/configcenter"
	"github.com/go-chassis/go-archaius/sources/enviromentvariable-source"
	"github.com/go-chassis/go-archaius/sources/file-source"
	"github.com/go-chassis/go-archaius/sources/memory-source"
	"github.com/go-chassis/go-chassis-config"
	"github.com/go-mesh/openlogging"
)

var (
	factory ConfigurationFactory
	fs      filesource.FileSource
	ms      = memoryconfigsource.NewMemoryConfigurationSource()

	running             = false
	configServerRunning = false
)

func initFileSource(o *Options) (core.ConfigSource, error) {
	files := make([]string, 0)
	// created file source object
	fs = filesource.NewFileSource()
	// adding all files with file source
	for _, v := range o.RequiredFiles {
		if err := fs.AddFile(v, filesource.DefaultFilePriority, o.FileHandler); err != nil {
			openlogging.GetLogger().Errorf("add file source error [%s].", err.Error())
			return nil, err
		}
		files = append(files, v)
	}
	for _, v := range o.OptionalFiles {
		_, err := os.Stat(v)
		if os.IsNotExist(err) {
			openlogging.GetLogger().Infof("[%s] not exist", v)
			continue
		}
		if err := fs.AddFile(v, filesource.DefaultFilePriority, o.FileHandler); err != nil {
			openlogging.GetLogger().Infof("%v", err)
			return nil, err
		}
		files = append(files, v)
	}
	openlogging.GetLogger().Infof("Configuration files: %s", strings.Join(files, ", "))
	return fs, nil
}

// Init create a Archaius config singleton
func Init(opts ...Option) error {
	if running {
		openlogging.Debug("can not init archaius again, call Clean first")
		return nil
	}
	var err error
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	// created config factory object
	factory, err = NewConfigFactory()
	if err != nil {

		return err
	}
	factory.DeInit()
	factory.Init()

	fs, err := initFileSource(o)
	if err != nil {
		return err
	}
	if o.ConfigCenterInfo != (ConfigCenterInfo{}) {
		if err := EnableConfigCenterSource(o.ConfigCenterInfo, o.ConfigClient); err != nil {
			return err
		}
	}
	err = factory.AddSource(fs)
	if err != nil {
		return err
	}

	// build-in config sources
	if o.UseMemSource {
		ms = memoryconfigsource.NewMemoryConfigurationSource()
		factory.AddSource(ms)
	}
	if o.UseCLISource {
		cmdSource := commandlinesource.NewCommandlineConfigSource()
		factory.AddSource(cmdSource)
	}
	if o.UseENVSource {
		envSource := envconfigsource.NewEnvConfigurationSource()
		factory.AddSource(envSource)
	}

	eventHandler := EventListener{
		Name:    "EventHandler",
		Factory: factory,
	}

	factory.RegisterListener(eventHandler, "a*")
	openlogging.GetLogger().Info("archaius init success")
	running = true
	return nil
}

//CustomInit accept is able to accept a list of config source, add it into archaius runtime.
//it almost like Init(), but you can fully control config sources you inject to archaius
func CustomInit(sources ...core.ConfigSource) error {
	var err error
	factory, err = NewConfigFactory()
	if err != nil {
		return err
	}

	factory.DeInit()
	factory.Init()
	for _, s := range sources {
		err = factory.AddSource(s)
		if err != nil {
			return err
		}
	}
	return err
}

//EnableConfigCenterSource create a config center source singleton
//A config center source pull remote config server key values into local memory
//so that you can use GetXXX to get value easily
func EnableConfigCenterSource(ci ConfigCenterInfo, cc config.Client) error {
	if ci == (ConfigCenterInfo{}) {
		return errors.New("ConfigCenterInfo can not be empty")
	}
	if configServerRunning {
		openlogging.Warn("can not init config server again, call Clean first")
		return nil
	}

	var err error
	if cc == nil {
		opts := config.Options{
			ServerURI:     ci.URL,
			TenantName:    ci.TenantName,
			EnableSSL:     ci.EnableSSL,
			TLSConfig:     ci.TLSConfig,
			RefreshPort:   ci.RefreshPort,
			AutoDiscovery: ci.AutoDiscovery,

			Version:     ci.Version,
			ServiceName: ci.Service,
			App:         ci.App,
			Env:         ci.Environment,
		}
		cc, err = config.NewClient(ci.ClientType, opts)
		if err != nil {
			return err
		}
	}
	configCenterSource := configcenter.NewConfigCenterSource(cc, ci.RefreshMode,
		ci.RefreshInterval)
	err = factory.AddSource(configCenterSource)
	if err != nil {
		return err
	}
	configServerRunning = true
	return nil
}

// EventListener is a struct having information about registering key and object
type EventListener struct {
	Name    string
	Factory ConfigurationFactory
}

// Event is invoked while generating events at run time
func (e EventListener) Event(event *core.Event) {
	value := e.Factory.GetConfigurationByKey(event.Key)
	openlogging.GetLogger().Debugf("config value after change %s | %s", event.Key, value)
}

// Get is for to get the value of configuration key
func Get(key string) interface{} {
	return factory.GetConfigurationByKey(key)
}

// Exist check the configuration key existence
func Exist(key string) bool {
	return factory.IsKeyExist(key)
}

// UnmarshalConfig is for unmarshalling the configuraions of receiving object
func UnmarshalConfig(obj interface{}) error {
	return factory.Unmarshal(obj)
}

// GetBool is gives the key value in the form of bool
func GetBool(key string, defaultValue bool) bool {
	b, err := factory.GetValue(key).ToBool()
	if err != nil {
		return defaultValue
	}
	return b
}

// GetFloat64 gives the key value in the form of float64
func GetFloat64(key string, defaultValue float64) float64 {
	result, err := factory.GetValue(key).ToFloat64()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetInt gives the key value in the form of GetInt
func GetInt(key string, defaultValue int) int {
	result, err := factory.GetValue(key).ToInt()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetString gives the key value in the form of GetString
func GetString(key string, defaultValue string) string {
	result, err := factory.GetValue(key).ToString()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetConfigs gives the information about all configurations
func GetConfigs() map[string]interface{} {
	return factory.GetConfigurations()
}

// GetStringByDI get the value of configuration key in other dimension
func GetStringByDI(dimensionInfo, key string, defaultValue string) string {
	result, err := factory.GetValueByDI(dimensionInfo, key).ToString()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetConfigsByDI get the all configurations in other dimension
func GetConfigsByDI(dimensionInfo string) map[string]interface{} {
	return factory.GetConfigurationsByDimensionInfo(dimensionInfo)
}

// AddDI adds a NewDimensionInfo of which configurations needs to be taken
func AddDI(dimensionInfo string) (map[string]string, error) {
	config, err := factory.AddByDimensionInfo(dimensionInfo)
	return config, err
}

//RegisterListener to Register all listener for different key changes, each key could be a regular expression
func RegisterListener(listenerObj core.EventListener, key ...string) error {
	return factory.RegisterListener(listenerObj, key...)
}

// UnRegisterListener is to remove the listener
func UnRegisterListener(listenerObj core.EventListener, key ...string) error {
	return factory.UnRegisterListener(listenerObj, key...)
}

// AddFile is for to add the configuration files into the configfactory at run time
func AddFile(file string, opts ...FileOption) error {
	o := &FileOptions{}
	for _, f := range opts {
		f(o)
	}
	if err := fs.AddFile(file, filesource.DefaultFilePriority, o.Handler); err != nil {
		return err
	}
	return factory.Refresh(fs.GetSourceName())
}

//AddKeyValue add the configuration key, value pairs into memory source at runtime
//it is just affect the local configs
func AddKeyValue(key string, value interface{}) error {
	return ms.AddKeyValue(key, value)
}

// DeleteKeyValue delete the configuration key, value pairs in memory source
func DeleteKeyValue(key string, value interface{}) error {
	return ms.DeleteKeyValue(key, value)
}

//AddSource add source implementation
func AddSource(source core.ConfigSource) error {
	return factory.AddSource(source)
}

//GetConfigFactory return factory
func GetConfigFactory() ConfigurationFactory {
	return factory
}

//Clean will call config manager CleanUp Method,
//it deletes all sources which means all of key value is deleted.
//after you call Clean, you can init archaius again
func Clean() error {
	err := factory.DeInit()
	if err != nil {
		return err
	}
	running = false
	configServerRunning = false
	return nil
}
