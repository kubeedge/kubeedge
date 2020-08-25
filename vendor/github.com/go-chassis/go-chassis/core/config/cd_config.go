package config

import "github.com/go-chassis/go-archaius"

// GetContractDiscoveryType returns the Type of contract discovery registry
func GetContractDiscoveryType() string {
	if GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Type != "" {
		return GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Type
	}
	return GlobalDefinition.Cse.Service.Registry.Type
}

// GetContractDiscoveryAddress returns the Address of contract discovery registry
func GetContractDiscoveryAddress() string {
	if GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Address != "" {
		return GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Address
	}
	return GlobalDefinition.Cse.Service.Registry.Address
}

// GetContractDiscoveryTenant returns the Tenant of contract discovery registry
func GetContractDiscoveryTenant() string {
	if GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Tenant != "" {
		return GlobalDefinition.Cse.Service.Registry.ContractDiscovery.Tenant
	}
	return GlobalDefinition.Cse.Service.Registry.Tenant
}

// GetContractDiscoveryAPIVersion returns the APIVersion of contract discovery registry
func GetContractDiscoveryAPIVersion() string {
	if GlobalDefinition.Cse.Service.Registry.ContractDiscovery.APIVersion.Version != "" {
		return GlobalDefinition.Cse.Service.Registry.ContractDiscovery.APIVersion.Version
	}
	return GlobalDefinition.Cse.Service.Registry.APIVersion.Version
}

// GetContractDiscoveryDisable returns the Disable of contract discovery registry
func GetContractDiscoveryDisable() bool {
	if b := archaius.GetBool("cse.service.registry.contractDiscovery.disabled", false); b {
		return b
	}
	return archaius.GetBool("cse.service.registry.disabled", false)
}
