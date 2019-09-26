package registry

import (
	"errors"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/schema"
	"github.com/go-chassis/go-chassis/core/metadata"
	"github.com/go-chassis/go-chassis/pkg/runtime"
	"github.com/go-mesh/openlogging"
)

var errEmptyServiceIDFromRegistry = errors.New("got empty serviceID from registry")

// microServiceDependencies micro-service dependencies
var microServiceDependencies *MicroServiceDependency

// InstanceEndpoints instance endpoints
var InstanceEndpoints = make(map[string]string)

// RegisterMicroservice register micro-service
func RegisterMicroservice() error {
	service := config.MicroserviceDefinition
	if e := service.ServiceDescription.Environment; e != "" {
		openlogging.GetLogger().Infof("Microservice environment: [%s]", e)
	} else {
		openlogging.Debug("No microservice environment defined")
	}
	microServiceDependencies = &MicroServiceDependency{}
	var err error
	runtime.Schemas, err = schema.GetSchemaIDs(service.ServiceDescription.Name)
	if err != nil {
		openlogging.GetLogger().Warnf("No schemas file for microservice [%s].", service.ServiceDescription.Name)
		runtime.Schemas = make([]string, 0)
	}
	if service.ServiceDescription.Level == "" {
		service.ServiceDescription.Level = common.DefaultLevel
	}
	if service.ServiceDescription.ServicesStatus == "" {
		service.ServiceDescription.ServicesStatus = common.DefaultStatus
	}
	if service.ServiceDescription.Properties == nil {
		service.ServiceDescription.Properties = make(map[string]string)
	}
	framework := metadata.NewFramework()

	svcPaths := service.ServiceDescription.ServicePaths
	var regpaths []ServicePath
	for _, svcPath := range svcPaths {
		var regpath ServicePath
		regpath.Path = svcPath.Path
		regpath.Property = svcPath.Property
		regpaths = append(regpaths, regpath)
	}
	microservice := &MicroService{
		ServiceID:   runtime.ServiceID,
		AppID:       runtime.App,
		ServiceName: service.ServiceDescription.Name,
		Version:     service.ServiceDescription.Version,
		Paths:       regpaths,
		Environment: service.ServiceDescription.Environment,
		Status:      service.ServiceDescription.ServicesStatus,
		Level:       service.ServiceDescription.Level,
		Schemas:     runtime.Schemas,
		Framework: &Framework{
			Version: framework.Version,
			Name:    framework.Name,
		},
		RegisterBy: framework.Register,
		Metadata:   make(map[string]string),
		// TODO allows to customize microservice alias
		Alias: "",
	}
	//update metadata
	if len(microservice.Alias) == 0 {
		// if the microservice is allowed to be called by consumers with different appId,
		// this means that the governance configuration of the consumer side needs to
		// support key format with appid, like 'cse.loadbalance.{alias}.strategy.name'.
		microservice.Alias = microservice.AppID + ":" + microservice.ServiceName
	}
	if config.GetRegistratorScope() == common.ScopeFull {
		microservice.Metadata["allowCrossApp"] = common.TRUE
		service.ServiceDescription.Properties["allowCrossApp"] = common.TRUE
	} else {
		service.ServiceDescription.Properties["allowCrossApp"] = common.FALSE
	}
	openlogging.GetLogger().Debugf("Update micro service properties%v", service.ServiceDescription.Properties)
	openlogging.GetLogger().Infof("Framework registered is [ %s:%s ]", framework.Name, framework.Version)
	openlogging.GetLogger().Infof("Micro service registered by [ %s ]", framework.Register)

	sid, err := DefaultRegistrator.RegisterService(microservice)
	if err != nil {
		openlogging.GetLogger().Errorf("Register [%s] failed: %s", microservice.ServiceName, err)
		return err
	}
	if sid == "" {
		openlogging.Error(errEmptyServiceIDFromRegistry.Error())
		return errEmptyServiceIDFromRegistry
	}
	runtime.ServiceID = sid
	openlogging.GetLogger().Infof("Register [%s/%s] success", runtime.ServiceID, microservice.ServiceName)

	return nil
}

// RegisterMicroserviceInstances register micro-service instances
func RegisterMicroserviceInstances() error {
	var err error
	service := config.MicroserviceDefinition
	runtime.Schemas, err = schema.GetSchemaIDs(service.ServiceDescription.Name)
	for _, schemaID := range runtime.Schemas {
		schemaInfo := schema.DefaultSchemaIDsMap[schemaID]
		err := DefaultRegistrator.AddSchemas(runtime.ServiceID, schemaID, schemaInfo)
		if err != nil {
			openlogging.Warn("upload contract to registry failed: " + err.Error())
		}
		openlogging.Info("upload schema to registry, " + schemaID)
	}

	openlogging.Info("Start to register instance.")

	sid, err := DefaultServiceDiscoveryService.GetMicroServiceID(runtime.App, service.ServiceDescription.Name, service.ServiceDescription.Version, service.ServiceDescription.Environment)
	if err != nil {
		openlogging.GetLogger().Errorf("Get service failed, key: %s:%s:%s, err %s",
			runtime.App,
			service.ServiceDescription.Name,
			service.ServiceDescription.Version, err)
		return err
	}
	eps, err := MakeEndpointMap(config.GlobalDefinition.Cse.Protocols)
	if err != nil {
		return err
	}
	openlogging.GetLogger().Infof("service support protocols %v", config.GlobalDefinition.Cse.Protocols)
	if len(InstanceEndpoints) != 0 {
		eps = InstanceEndpoints
	}
	if service.ServiceDescription.ServicesStatus == "" {
		service.ServiceDescription.ServicesStatus = common.DefaultStatus
	}
	microServiceInstance := &MicroServiceInstance{
		EndpointsMap: eps,
		HostName:     runtime.HostName,
		Status:       service.ServiceDescription.ServicesStatus,
		Metadata:     map[string]string{"nodeIP": config.NodeIP},
	}

	var dInfo = new(DataCenterInfo)
	if config.GlobalDefinition.DataCenter.Name != "" && config.GlobalDefinition.DataCenter.AvailableZone != "" {
		dInfo.Name = config.GlobalDefinition.DataCenter.Name
		dInfo.Region = config.GlobalDefinition.DataCenter.Name
		dInfo.AvailableZone = config.GlobalDefinition.DataCenter.AvailableZone
		microServiceInstance.DataCenterInfo = dInfo
	}

	instanceID, err := DefaultRegistrator.RegisterServiceInstance(sid, microServiceInstance)
	if err != nil {
		openlogging.GetLogger().Errorf("Register instance failed, serviceID: %s, err %s", sid, err.Error())
		return err
	}
	//Set to runtime
	runtime.InstanceID = instanceID
	runtime.InstanceStatus = runtime.StatusRunning
	if service.ServiceDescription.InstanceProperties != nil {
		if err := DefaultRegistrator.UpdateMicroServiceInstanceProperties(sid, instanceID, service.ServiceDescription.InstanceProperties); err != nil {
			openlogging.GetLogger().Errorf("UpdateMicroServiceInstanceProperties failed, microServiceID/instanceID = %s/%s.", sid, instanceID)
			return err
		}
		runtime.InstanceMD = service.ServiceDescription.InstanceProperties
		openlogging.GetLogger().Debugf("UpdateMicroServiceInstanceProperties success, microServiceID/instanceID = %s/%s.", sid, instanceID)
	}

	value, _ := SelfInstancesCache.Get(microServiceInstance.ServiceID)
	instanceIDs, _ := value.([]string)
	var isRepeat bool
	for _, va := range instanceIDs {
		if va == instanceID {
			isRepeat = true
		}
	}
	if !isRepeat {
		instanceIDs = append(instanceIDs, instanceID)
	}
	SelfInstancesCache.Set(sid, instanceIDs, 0)
	openlogging.GetLogger().Infof("Register instance success, serviceID/instanceID: %s/%s.", sid, instanceID)
	return nil
}
