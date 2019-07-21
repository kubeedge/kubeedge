package registry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/lager"
	"github.com/go-mesh/openlogging"
)

// constant values for registry parameters
const (
	DefaultRegistratorPlugin       = "servicecenter"
	DefaultServiceDiscoveryPlugin  = "servicecenter"
	DefaultContractDiscoveryPlugin = "servicecenter"
	Name                           = "registry"
	SDTag                          = "serviceDiscovery"
	CDTag                          = "contractDiscovery"
	RTag                           = "registrator"
	Auto                           = "auto"
	Manual                         = "manual"
)

// IsEnabled check enable
var IsEnabled bool
var mu sync.Mutex

// DefaultRegistrator is the client of registry, you can call the method of it to interact with microservice registry
var DefaultRegistrator Registrator

// DefaultAddr default address of service center
var DefaultAddr = "http://127.0.0.1:30100"

// registryFunc registry function
var registryFunc = make(map[string]func(opts Options) Registrator)

// HBService variable of heartbeat service
var HBService = &HeartbeatService{
	instances: make(map[string]*HeartbeatTask),
}

// Registrator is the interface for developer to update information in service registry
type Registrator interface {
	//Close destroy connection between the registry client and server
	Close() error
	//RegisterService register a microservice to registry, if it is duplicated in registry, it returns error
	RegisterService(microService *MicroService) (string, error)
	//RegisterServiceInstance register a microservice instance to registry
	RegisterServiceInstance(sid string, instance *MicroServiceInstance) (string, error)
	RegisterServiceAndInstance(microService *MicroService, instance *MicroServiceInstance) (string, string, error)
	Heartbeat(microServiceID, microServiceInstanceID string) (bool, error)
	AddDependencies(dep *MicroServiceDependency) error
	UnRegisterMicroServiceInstance(microServiceID, microServiceInstanceID string) error
	UpdateMicroServiceInstanceStatus(microServiceID, microServiceInstanceID, status string) error
	UpdateMicroServiceProperties(microServiceID string, properties map[string]string) error
	UpdateMicroServiceInstanceProperties(microServiceID, microServiceInstanceID string, properties map[string]string) error
	AddSchemas(microServiceID, schemaName, schemaInfo string) error
}

func enableRegistrator(opts Options) error {
	if config.GetRegistratorDisable() {
		return nil
	}

	rt := config.GetRegistratorType()
	if rt == "" {
		rt = DefaultRegistratorPlugin
	}
	var err error
	DefaultRegistrator, err = NewRegistrator(rt, opts)
	if err != nil {
		return err
	}

	if err := RegisterMicroservice(); err != nil {
		lager.Logger.Errorf("start backoff for register microservice: %s", err)
		startBackOff(RegisterMicroservice)
	}
	go HBService.Start()

	lager.Logger.Infof("Enable [%s] registrator.", rt)
	return nil
}

// InstallRegistrator install registrator plugin
func InstallRegistrator(name string, f func(opts Options) Registrator) {
	registryFunc[name] = f
	openlogging.Info("Installed registry plugin: " + name)
}

//NewRegistrator return registrator
func NewRegistrator(name string, opts Options) (Registrator, error) {
	f := registryFunc[name]
	if f == nil {
		return nil, fmt.Errorf("no registry plugin: %s", name)
	}
	return f(opts), nil
}
func getSpecifiedOptions() (oR, oSD, oCD Options, err error) {
	hostsR, schemeR, err := URIs2Hosts(strings.Split(config.GetRegistratorAddress(), ","))
	if err != nil {
		return
	}
	oR.Addrs = hostsR
	oR.Tenant = config.GetRegistratorTenant()
	oR.Version = config.GetRegistratorAPIVersion()
	oR.TLSConfig, err = getTLSConfig(schemeR, RTag)
	if err != nil {
		return
	}
	if oR.TLSConfig != nil {
		oR.EnableSSL = true
	}
	hostsSD, schemeSD, err := URIs2Hosts(strings.Split(config.GetServiceDiscoveryAddress(), ","))
	if err != nil {
		return
	}
	oSD.Addrs = hostsSD
	oSD.Tenant = config.GetServiceDiscoveryTenant()
	oSD.Version = config.GetServiceDiscoveryAPIVersion()
	oSD.ConfigPath = config.GetServiceDiscoveryConfigPath()
	oSD.TLSConfig, err = getTLSConfig(schemeSD, SDTag)
	if err != nil {
		return
	}
	if oSD.TLSConfig != nil {
		oSD.EnableSSL = true
	}
	hostsCD, schemeCD, err := URIs2Hosts(strings.Split(config.GetContractDiscoveryAddress(), ","))
	if err != nil {
		return
	}
	oCD.Addrs = hostsCD
	oCD.Tenant = config.GetContractDiscoveryTenant()
	oCD.Version = config.GetContractDiscoveryAPIVersion()
	oCD.TLSConfig, err = getTLSConfig(schemeCD, CDTag)
	if err != nil {
		return
	}
	if oCD.TLSConfig != nil {
		oCD.EnableSSL = true
	}
	return
}

// Enable create DefaultRegistrator
func Enable() (err error) {
	mu.Lock()
	defer mu.Unlock()
	if IsEnabled {
		return
	}

	var oR, oSD, oCD Options
	if oR, oSD, oCD, err = getSpecifiedOptions(); err != nil {
		return err
	}

	enableRegistryCache()
	if err := enableRegistrator(oR); err != nil {
		return err
	}
	if err := enableServiceDiscovery(oSD); err != nil {
		return err
	}
	enableContractDiscovery(oCD)

	lager.Logger.Info("Enabled Registry")
	IsEnabled = true
	return nil
}

// DoRegister for registering micro-service instances
func DoRegister() error {
	var (
		isAutoRegister bool
		t              = config.GetRegistratorAutoRegister()
	)
	switch t {
	case "":
		isAutoRegister = true
	case Auto:
		isAutoRegister = true
	case Manual:
		isAutoRegister = false
	default:
		{
			tmpErr := fmt.Errorf("parameter incorrect, autoregister: %s", t)
			lager.Logger.Error(tmpErr.Error())
			return tmpErr
		}
	}
	if isAutoRegister {
		if err := RegisterMicroserviceInstances(); err != nil {
			lager.Logger.Errorf("start back off for register microservice instances background: %s", err)
			go startBackOff(RegisterMicroserviceInstances)
		}
	}
	return nil
}
