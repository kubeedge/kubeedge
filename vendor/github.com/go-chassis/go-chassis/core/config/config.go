package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/config/schema"
	"github.com/go-chassis/go-chassis/pkg/runtime"
	"github.com/go-chassis/go-chassis/pkg/util/fileutil"
	"github.com/go-chassis/go-chassis/pkg/util/iputil"
	"github.com/go-mesh/openlogging"
	"gopkg.in/yaml.v2"
)

// GlobalDefinition is having the information about region, load balancing, service center, config center,
// protocols, and handlers for the micro service
var GlobalDefinition *model.GlobalCfg
var lbConfig *model.LBWrapper

// MicroserviceDefinition is having the info about application id, provider info, description of the service,
// and description of the instance
var MicroserviceDefinition *model.MicroserviceCfg

// RouterDefinition is route rule config
var RouterDefinition *model.RouterConfig

//HystrixConfig is having info about isolation, circuit breaker, fallback properities of the micro service
var HystrixConfig *model.HystrixConfigWrapper

// NodeIP gives the information of node ip
var NodeIP string

// SelfServiceName is self micro service name
//Deprecated, plz use runtime.ServiceName
var SelfServiceName string

// SelfVersion gives version of the self micro service
//Deprecated, use runtime pkg
var SelfVersion string

// ErrNoName is used to represent the service name missing error
var ErrNoName = errors.New("micro service name is missing in description file")

//GetConfigCenterConf return config center conf
func GetConfigCenterConf() model.ConfigClient {
	return GlobalDefinition.Cse.Config.Client
}

//GetTransportConf return transport settings
func GetTransportConf() model.Transport {
	return GlobalDefinition.Cse.Transport
}

//GetDataCenter return data center info
func GetDataCenter() *model.DataCenterInfo {
	return GlobalDefinition.DataCenter
}

// parse unmarshal configurations on respective structure
func parse() error {
	err := ReadGlobalConfigFile()
	if err != nil {
		return err
	}
	err = ReadLBFromArchaius()
	if err != nil {
		return err
	}

	err = ReadHystrixFromArchaius()
	if err != nil {
		return err
	}

	err = readMicroserviceConfigFiles()
	if err != nil {
		return err
	}

	populateConfigCenterAddress()
	populateServiceRegistryAddress()
	populateMonitorServerAddress()
	populateServiceEnvironment()
	populateServiceName()
	populateVersion()
	populateTenant()

	return nil
}

// populateServiceRegistryAddress populate service registry address
func populateServiceRegistryAddress() {
	//Registry Address , higher priority for environment variable
	registryAddrFromEnv := os.Getenv(common.EnvCSEEndpoint)
	if registryAddrFromEnv == "" {
		registryAddrFromEnv = archaius.GetString(common.CseRegistryAddress, "")
	}
	if registryAddrFromEnv != "" {
		GlobalDefinition.Cse.Service.Registry.Registrator.Address = registryAddrFromEnv
		GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Address = registryAddrFromEnv
		GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Address = registryAddrFromEnv
		GlobalDefinition.Cse.Service.Registry.Address = registryAddrFromEnv
	}

}

// populateConfigCenterAddress populate config center address
func populateConfigCenterAddress() {
	//Config Center Address , higher priority for environment variable
	configCenterAddrFromEnv := os.Getenv(common.EnvCSEEndpoint)
	if configCenterAddrFromEnv == "" {
		configCenterAddrFromEnv = archaius.GetString(common.CseConfigCenterAddress, "")
	}
	if configCenterAddrFromEnv != "" {
		GlobalDefinition.Cse.Config.Client.ServerURI = configCenterAddrFromEnv
	}
}

// populateMonitorServerAddress populate monitor server address
func populateMonitorServerAddress() {
	//Monitor Center Address , higher priority for environment variable
	monitorServerAddrFromEnv := archaius.GetString(common.CseMonitorServer, "")
	if monitorServerAddrFromEnv != "" {
		GlobalDefinition.Cse.Monitor.Client.ServerURI = monitorServerAddrFromEnv
	}
}

// populateServiceEnvironment populate service environment
func populateServiceEnvironment() {
	if e := archaius.GetString(common.Env, ""); e != "" {
		MicroserviceDefinition.ServiceDescription.Environment = e
	}
}

// populateServiceName populate service name
func populateServiceName() {
	if e := archaius.GetString(common.ServiceName, ""); e != "" {
		MicroserviceDefinition.ServiceDescription.Name = e
	}
}

// populateVersion populate version
func populateVersion() {
	if e := archaius.GetString(common.Version, ""); e != "" {
		MicroserviceDefinition.ServiceDescription.Version = e
	}
}

// populateTenant populate tenant
func populateTenant() {
	if GlobalDefinition.Cse.Service.Registry.Tenant == "" {
		GlobalDefinition.Cse.Service.Registry.Tenant = common.DefaultApp
	}
}

// ReadGlobalConfigFile for to unmarshal the global config file(chassis.yaml) information
func ReadGlobalConfigFile() error {
	globalDef := model.GlobalCfg{}
	err := archaius.UnmarshalConfig(&globalDef)
	if err != nil {
		return err
	}
	GlobalDefinition = &globalDef
	return nil
}

// ReadLBFromArchaius for to unmarshal the global config file(chassis.yaml) information
func ReadLBFromArchaius() error {
	lbMutex.Lock()
	defer lbMutex.Unlock()
	lbDef := model.LBWrapper{}
	err := archaius.UnmarshalConfig(&lbDef)
	if err != nil {
		return err
	}
	lbConfig = &lbDef

	return nil
}

type pathError struct {
	Path string
	Err  error
}

func (e *pathError) Error() string { return e.Path + ": " + e.Err.Error() }

// parseRouterConfig is unmarshal the router configuration file(router.yaml)
func parseRouterConfig(file string) error {
	RouterDefinition = &model.RouterConfig{}
	err := unmarshalYamlFile(file, RouterDefinition)
	if err != nil && !os.IsNotExist(err) {
		return &pathError{Path: file, Err: err}
	}
	return err
}

func unmarshalYamlFile(file string, target interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, target)
}

// ReadHystrixFromArchaius is unmarshal hystrix configuration file(circuit_breaker.yaml)
func ReadHystrixFromArchaius() error {
	cbMutex.RLock()
	defer cbMutex.RUnlock()
	hystrixCnf := model.HystrixConfigWrapper{}
	err := archaius.UnmarshalConfig(&hystrixCnf)
	if err != nil {
		return err
	}
	HystrixConfig = &hystrixCnf
	return nil
}

// readMicroserviceConfigFiles read micro service configuration file
func readMicroserviceConfigFiles() error {
	MicroserviceDefinition = &model.MicroserviceCfg{}
	//find only one microservice yaml
	microserviceNames := schema.GetMicroserviceNames()
	defPath := fileutil.GetMicroserviceDesc()
	data, err := ioutil.ReadFile(defPath)
	if err != nil {
		openlogging.GetLogger().Errorf(fmt.Sprintf("WARN: Missing microservice description file: %s", err.Error()))
		if len(microserviceNames) == 0 {
			return errors.New("missing microservice description file")
		}
		msName := microserviceNames[0]
		msDefPath := fileutil.MicroserviceDefinition(msName)
		openlogging.GetLogger().Warnf(fmt.Sprintf("Try to find microservice description file in [%s]", msDefPath))
		data, err := ioutil.ReadFile(msDefPath)
		if err != nil {
			return fmt.Errorf("missing microservice description file: %s", err.Error())
		}
		ReadMicroserviceConfigFromBytes(data)
		return nil
	}
	return ReadMicroserviceConfigFromBytes(data)
}

// ReadMicroserviceConfigFromBytes read micro service configurations from bytes
func ReadMicroserviceConfigFromBytes(data []byte) error {
	microserviceDef := model.MicroserviceCfg{}
	err := yaml.Unmarshal([]byte(data), &microserviceDef)
	if err != nil {
		return err
	}
	if microserviceDef.ServiceDescription.Name == "" {
		return ErrNoName
	}
	if microserviceDef.ServiceDescription.Version == "" {
		microserviceDef.ServiceDescription.Version = common.DefaultVersion
	}

	MicroserviceDefinition = &microserviceDef
	return nil
}

//GetLoadBalancing return lb config
func GetLoadBalancing() *model.LoadBalancing {
	if lbConfig != nil {
		return lbConfig.Prefix.LBConfig
	}
	return nil
}

//GetHystrixConfig return cb config
func GetHystrixConfig() *model.HystrixConfig {
	return HystrixConfig.HystrixConfig
}

// Init is initialize the configuration directory, archaius, route rule, and schema
func Init() error {
	if err := parseRouterConfig(fileutil.RouterDefinition()); err != nil {
		if os.IsNotExist(err) {
			openlogging.GetLogger().Infof("[%s] not exist", fileutil.RouterDefinition())
		} else {
			return err
		}
	}
	err := InitArchaius()
	if err != nil {
		return err
	}
	openlogging.GetLogger().Infof("archaius init success")

	//Upload schemas using environment variable SCHEMA_ROOT
	schemaPath := archaius.GetString(common.EnvSchemaRoot, "")
	if schemaPath == "" {
		schemaPath = fileutil.GetConfDir()
	}

	schemaError := schema.LoadSchema(schemaPath)
	if schemaError != nil {
		return schemaError
	}

	//set microservice names
	msError := schema.SetMicroServiceNames(schemaPath)
	if msError != nil {
		return msError
	}

	NodeIP = archaius.GetString(common.EnvNodeIP, "")
	err = parse()
	if err != nil {
		return err
	}

	SelfServiceName = MicroserviceDefinition.ServiceDescription.Name
	runtime.ServiceName = MicroserviceDefinition.ServiceDescription.Name
	SelfVersion = MicroserviceDefinition.ServiceDescription.Version
	runtime.Version = MicroserviceDefinition.ServiceDescription.Version
	runtime.Environment = MicroserviceDefinition.ServiceDescription.Environment
	runtime.MD = MicroserviceDefinition.ServiceDescription.Properties
	if MicroserviceDefinition.AppID != "" { //microservice.yaml has first priority
		runtime.App = MicroserviceDefinition.AppID
	} else if GlobalDefinition.AppID != "" { //chassis.yaml has second priority
		runtime.App = GlobalDefinition.AppID
	}
	if runtime.App == "" {
		runtime.App = common.DefaultApp
	}

	runtime.HostName = MicroserviceDefinition.ServiceDescription.Hostname
	if runtime.HostName == "" {
		runtime.HostName, err = os.Hostname()
		if err != nil {
			openlogging.Error("Get hostname failed:" + err.Error())
			return err
		}
	} else if runtime.HostName == common.PlaceholderInternalIP {
		runtime.HostName = iputil.GetLocalIP()
	}
	openlogging.Info("Host name is " + runtime.HostName)
	return err
}
