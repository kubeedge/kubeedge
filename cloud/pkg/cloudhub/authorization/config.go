/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package authorization

import (
	"fmt"

	"k8s.io/apiserver/pkg/apis/apiserver"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/client-go/informers"
	"k8s.io/kubernetes/pkg/auth/nodeidentifier"
	apiserverauthorizer "k8s.io/kubernetes/pkg/kubeapiserver/authorizer"
	"k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/node"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac/bootstrappolicy"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	cloudhubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
)

// Authorizer provides authorization enhancements for CloudHub
type Authorizer interface {
	// AdmitMessage determines whether the message should be admitted
	AdmitMessage(message beehivemodel.Message, info cloudhubmodel.HubInfo) error
	// AuthenticateConnection authenticates the new connection
	AuthenticateConnection(connection conn.Connection) error
}

// Config Authorizer's configurations
type Config struct {
	Enabled bool
	// Debug Authorizer logs errors but always allows messages
	Debug                    bool
	AuthorizationModes       []string
	VersionedInformerFactory informers.SharedInformerFactory
}

// New creates new Authorizer
func (c Config) New() (Authorizer, error) {
	authorizers := make([]apiserver.AuthorizerConfiguration, 0, len(c.AuthorizationModes))
	for _, mode := range c.AuthorizationModes {
		authorizers = append(authorizers, apiserver.AuthorizerConfiguration{
			Type: apiserver.AuthorizerType(mode),
		})
	}
	config := apiserverauthorizer.Config{
		VersionedInformerFactory: c.VersionedInformerFactory,
		AuthorizationConfiguration: &apiserver.AuthorizationConfiguration{
			Authorizers: authorizers,
		},
	}
	authz, _, err := assembleAuthorizer(config)
	if err != nil {
		return nil, err
	}
	return &cloudhubAuthorizer{
		enabled: c.Enabled,
		debug:   c.Debug,
		authz:   authz,
	}, nil
}

func assembleAuthorizer(config apiserverauthorizer.Config) (authorizer.Authorizer, authorizer.RuleResolver, error) {
	if len(config.AuthorizationConfiguration.Authorizers) == 0 {
		return nil, nil, fmt.Errorf("at least one authorization mode must be passed")
	}

	var (
		authorizers   []authorizer.Authorizer
		ruleResolvers []authorizer.RuleResolver
	)

	for _, authzConfig := range config.AuthorizationConfiguration.Authorizers {
		// Keep cases in sync with constant list in k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes/modes.go.
		switch authzConfig.Type {
		case apiserver.AuthorizerType(modes.ModeNode):
			node.RegisterMetrics()
			graph := node.NewGraph()
			node.AddGraphEventHandlers(
				graph,
				config.VersionedInformerFactory.Core().V1().Nodes(),
				config.VersionedInformerFactory.Core().V1().Pods(),
				config.VersionedInformerFactory.Core().V1().PersistentVolumes(),
				config.VersionedInformerFactory.Storage().V1().VolumeAttachments(),
				nil,
			)
			nodeAuthorizer := node.NewAuthorizer(graph, nodeidentifier.NewDefaultNodeIdentifier(), bootstrappolicy.NodeRules())
			authorizers = append(authorizers, nodeAuthorizer)
			ruleResolvers = append(ruleResolvers, nodeAuthorizer)

		case apiserver.AuthorizerType(modes.ModeAlwaysAllow):
			alwaysAllowAuthorizer := authorizerfactory.NewAlwaysAllowAuthorizer()
			authorizers = append(authorizers, alwaysAllowAuthorizer)
			ruleResolvers = append(ruleResolvers, alwaysAllowAuthorizer)
		case apiserver.AuthorizerType(modes.ModeAlwaysDeny):
			alwaysDenyAuthorizer := authorizerfactory.NewAlwaysDenyAuthorizer()
			authorizers = append(authorizers, alwaysDenyAuthorizer)
			ruleResolvers = append(ruleResolvers, alwaysDenyAuthorizer)
		default:
			return nil, nil, fmt.Errorf("unknown authorization mode %s specified", authzConfig.Type)
		}
	}

	// put kubeedgeResourceAuthorizer at the tail of authorizer chain to allow kubeedge custom messages
	authorizers = append(authorizers, &kubeedgeResourceAuthorizer{})
	ruleResolvers = append(ruleResolvers, &kubeedgeResourceAuthorizer{})

	return union.New(authorizers...), union.NewRuleResolvers(ruleResolvers...), nil
}
