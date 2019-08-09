package config

import "github.com/go-chassis/go-archaius"

// GetServiceDiscoveryType returns the Type of SD registry
func GetServiceDiscoveryType() string {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Type != "" {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Type
	}
	return GlobalDefinition.Cse.Service.Registry.Type
}

// GetServiceDiscoveryAddress returns the Address of SD registry
func GetServiceDiscoveryAddress() string {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Address != "" {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Address
	}
	return GlobalDefinition.Cse.Service.Registry.Address
}

// GetServiceDiscoveryRefreshInterval returns the RefreshInterval of SD registry
func GetServiceDiscoveryRefreshInterval() string {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.RefreshInterval != "" {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.RefreshInterval
	}
	return GlobalDefinition.Cse.Service.Registry.RefreshInterval
}

// GetServiceDiscoveryWatch returns the Watch of SD registry
func GetServiceDiscoveryWatch() bool {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Watch {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Watch
	}
	return GlobalDefinition.Cse.Service.Registry.Watch
}

// GetServiceDiscoveryTenant returns the Tenant of SD registry
func GetServiceDiscoveryTenant() string {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Tenant != "" {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.Tenant
	}
	return GlobalDefinition.Cse.Service.Registry.Tenant
}

// GetServiceDiscoveryAPIVersion returns the APIVersion of SD registry
func GetServiceDiscoveryAPIVersion() string {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.APIVersion.Version != "" {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.APIVersion.Version
	}
	return GlobalDefinition.Cse.Service.Registry.APIVersion.Version
}

// GetServiceDiscoveryDisable returns the Disable of SD registry
func GetServiceDiscoveryDisable() bool {
	if b := archaius.GetBool("cse.service.registry.serviceDiscovery.disabled", false); b {
		return b
	}
	return archaius.GetBool("cse.service.registry.disabled", false)
}

// GetServiceDiscoveryHealthCheck returns the HealthCheck of SD registry
func GetServiceDiscoveryHealthCheck() bool {
	if b := archaius.GetBool("cse.service.registry.serviceDiscovery.healthCheck", false); b {
		return b
	}
	return archaius.GetBool("cse.service.registry.healthCheck", false)
}

// DefaultConfigPath set the default config path
const DefaultConfigPath = "/etc/.kube/config"

// GetServiceDiscoveryConfigPath returns the configpath of SD registry
func GetServiceDiscoveryConfigPath() string {
	if GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.ConfigPath != "" {
		return GlobalDefinition.Cse.Service.Registry.ServiceDiscovery.ConfigPath
	}
	return DefaultConfigPath
}
