package config

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/core/common"
)

// GetRegistratorType returns the Type of service registry
func GetRegistratorType() string {
	if GlobalDefinition.Cse.Service.Registry.Registrator.Type != "" {
		return GlobalDefinition.Cse.Service.Registry.Registrator.Type
	}
	return GlobalDefinition.Cse.Service.Registry.Type
}

// GetRegistratorAddress returns the Address of service registry
func GetRegistratorAddress() string {
	if GlobalDefinition.Cse.Service.Registry.Registrator.Address != "" {
		return GlobalDefinition.Cse.Service.Registry.Registrator.Address
	}
	return GlobalDefinition.Cse.Service.Registry.Address
}

// GetRegistratorScope returns the Scope of service registry
func GetRegistratorScope() string {
	if GlobalDefinition.Cse.Service.Registry.Registrator.Scope == "" {
		GlobalDefinition.Cse.Service.Registry.Registrator.Scope = common.ScopeFull
	}
	return GlobalDefinition.Cse.Service.Registry.Scope
}

// GetRegistratorAutoRegister returns the AutoRegister of service registry
func GetRegistratorAutoRegister() string {
	if GlobalDefinition.Cse.Service.Registry.Registrator.AutoRegister != "" {
		return GlobalDefinition.Cse.Service.Registry.Registrator.AutoRegister
	}
	return GlobalDefinition.Cse.Service.Registry.AutoRegister
}

// GetRegistratorTenant returns the Tenant of service registry
func GetRegistratorTenant() string {
	if GlobalDefinition.Cse.Service.Registry.Registrator.Tenant != "" {
		return GlobalDefinition.Cse.Service.Registry.Registrator.Tenant
	}
	return GlobalDefinition.Cse.Service.Registry.Tenant
}

// GetRegistratorAPIVersion returns the APIVersion of service registry
func GetRegistratorAPIVersion() string {
	if GlobalDefinition.Cse.Service.Registry.Registrator.APIVersion.Version != "" {
		return GlobalDefinition.Cse.Service.Registry.Registrator.APIVersion.Version
	}
	return GlobalDefinition.Cse.Service.Registry.APIVersion.Version
}

// GetRegistratorDisable returns the Disable of service registry
func GetRegistratorDisable() bool {
	if b := archaius.GetBool("cse.service.registry.registrator.disabled", false); b {
		return b
	}
	return archaius.GetBool("cse.service.registry.disabled", false)
}
