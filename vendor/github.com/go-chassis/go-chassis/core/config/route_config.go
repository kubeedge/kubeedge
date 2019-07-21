package config

import "github.com/go-chassis/go-chassis/core/config/model"

//DefaultRouterType set the default router type
const DefaultRouterType = "cse"

// GetRouterType returns the type of router
func GetRouterType() string {
	if RouterDefinition.Router.Infra != "" {
		return RouterDefinition.Router.Infra
	}
	return DefaultRouterType
}

// GetRouterEndpoints returns the router address
func GetRouterEndpoints() string {
	return RouterDefinition.Router.Address
}

// GetRouterReference returns the router address
func GetRouterReference() map[string]model.ReferencesStruct {
	return GlobalDefinition.Cse.References
}
