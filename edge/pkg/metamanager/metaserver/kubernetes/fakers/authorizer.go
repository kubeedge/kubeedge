package fakers

import (
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
)

func NewAlwaysAllowAuthorizer() authorizer.Authorizer {
	return authorizerfactory.NewAlwaysAllowAuthorizer()
}
