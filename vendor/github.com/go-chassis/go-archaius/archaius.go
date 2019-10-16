// Package archaius provides you APIs which helps to manage files,
// remote config center configurations
package archaius

import (
	"errors"
	"github.com/go-chassis/go-archaius/cast"
	"os"
	"strings"

	"github.com/go-chassis/go-archaius/event"
	"github.com/go-chassis/go-archaius/source"
	"github.com/go-chassis/go-archaius/source/cli"
	"github.com/go-chassis/go-archaius/source/env"
	"github.com/go-chassis/go-archaius/source/file"
	"github.com/go-chassis/go-archaius/source/mem"
	"github.com/go-chassis/go-archaius/source/remote"
	"github.com/go-chassis/go-chassis-config"
	"github.com/go-mesh/openlogging"
)

var (
	manager             *source.Manager
	fs                  filesource.FileSource
	running             = false
	configServerRunning = false
)

func initFileSource(o *Options) (source.ConfigSource, error) {
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
		openlogging.Warn("can not init archaius again, call Clean first")
		return nil
	}
	var err error
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	manager = source.NewManager()

	fs, err := initFileSource(o)
	if err != nil {
		return err
	}
	err = manager.AddSource(fs, fs.GetPriority())
	if err != nil {
		return err
	}

	if o.RemoteInfo != nil {
		if err := EnableRemoteSource(o.RemoteInfo, o.ConfigClient); err != nil {
			return err
		}
	}

	// build-in config sources
	if o.UseMemSource {
		ms := mem.NewMemoryConfigurationSource()
		manager.AddSource(ms, ms.GetPriority())
	}
	if o.UseCLISource {
		cmdSource := cli.NewCommandlineConfigSource()
		manager.AddSource(cmdSource, cmdSource.GetPriority())
	}
	if o.UseENVSource {
		envSource := env.NewEnvConfigurationSource()
		manager.AddSource(envSource, envSource.GetPriority())
	}

	openlogging.Info("archaius init success")
	running = true
	return nil
}

//CustomInit accept a list of config source, add it into archaius runtime.
//it almost like Init(), but you can fully control config sources you inject to archaius
func CustomInit(sources ...source.ConfigSource) error {
	var err error
	manager = source.NewManager()
	for _, s := range sources {
		err = manager.AddSource(s, s.GetPriority())
		if err != nil {
			return err
		}
	}
	return err
}

//EnableRemoteSource create a remote source singleton
//A config center source pull remote config server key values into local memory
//so that you can use GetXXX to get value easily
func EnableRemoteSource(ci *RemoteInfo, cc config.Client) error {
	if ci == nil {
		return errors.New("RemoteInfo can not be empty")
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
			Labels:        ci.DefaultDimension,
		}
		cc, err = config.NewClient(ci.ClientType, opts)
		if err != nil {
			return err
		}
	}
	configCenterSource := remote.NewConfigCenterSource(cc, ci.RefreshMode,
		ci.RefreshInterval)
	err = manager.AddSource(configCenterSource, configCenterSource.GetPriority())
	if err != nil {
		return err
	}
	configServerRunning = true
	return nil
}

// Get is for to get the value of configuration key
func Get(key string) interface{} {
	return manager.GetConfig(key)
}

//GetValue return interface
func GetValue(key string) cast.Value {
	var confValue cast.Value
	val := manager.GetConfig(key)
	if val == nil {
		confValue = cast.NewValue(nil, errors.New("key does not exist"))
	} else {
		confValue = cast.NewValue(val, nil)
	}

	return confValue
}

// Exist check the configuration key existence
func Exist(key string) bool {
	return manager.IsKeyExist(key)
}

// UnmarshalConfig unmarshal the config of receiving object
func UnmarshalConfig(obj interface{}) error {
	return manager.Unmarshal(obj)
}

// GetBool is gives the key value in the form of bool
func GetBool(key string, defaultValue bool) bool {
	b, err := GetValue(key).ToBool()
	if err != nil {
		return defaultValue
	}
	return b
}

// GetFloat64 gives the key value in the form of float64
func GetFloat64(key string, defaultValue float64) float64 {
	result, err := GetValue(key).ToFloat64()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetInt gives the key value in the form of GetInt
func GetInt(key string, defaultValue int) int {
	result, err := GetValue(key).ToInt()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetString gives the key value in the form of GetString
func GetString(key string, defaultValue string) string {
	result, err := GetValue(key).ToString()
	if err != nil {
		return defaultValue
	}
	return result
}

// GetConfigs gives the information about all configurations
func GetConfigs() map[string]interface{} {
	return manager.Configs()
}

// AddDimensionInfo adds a NewDimensionInfo of which configurations needs to be taken
func AddDimensionInfo(labels map[string]string) (map[string]string, error) {
	config, err := manager.AddDimensionInfo(labels)
	return config, err
}

//RegisterListener to Register all listener for different key changes, each key could be a regular expression
func RegisterListener(listenerObj event.Listener, key ...string) error {
	return manager.RegisterListener(listenerObj, key...)
}

// UnRegisterListener is to remove the listener
func UnRegisterListener(listenerObj event.Listener, key ...string) error {
	return manager.UnRegisterListener(listenerObj, key...)
}

// AddFile is for to add the configuration files at runtime
func AddFile(file string, opts ...FileOption) error {
	o := &FileOptions{}
	for _, f := range opts {
		f(o)
	}
	if err := fs.AddFile(file, filesource.DefaultFilePriority, o.Handler); err != nil {
		return err
	}
	return manager.Refresh(fs.GetSourceName())
}

//Set add the configuration key, value pairs into memory source at runtime
//it is just affect the local configs
func Set(key string, value interface{}) error {
	return manager.Set(key, value)
}

// Delete delete the configuration key, value pairs in memory source
func Delete(key string) error {
	return manager.Delete(key)
}

//AddSource add source implementation
func AddSource(source source.ConfigSource) error {
	return manager.AddSource(source, source.GetPriority())
}

//Clean will call config manager CleanUp Method,
//it deletes all sources which means all of key value is deleted.
//after you call Clean, you can init archaius again
func Clean() error {
	manager.Cleanup()
	running = false
	return nil
}
