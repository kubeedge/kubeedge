package router

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/lager"
	chassisTLS "github.com/go-chassis/go-chassis/core/tls"
	"github.com/go-chassis/go-chassis/pkg/util/iputil"
	"github.com/go-chassis/go-chassis/pkg/util/tags"
	"github.com/go-mesh/openlogging"
)

// RouterTLS defines tls prefix
const RouterTLS = "router"

// Init initialize router config
func Init() error {
	// init dests and templates
	routerConfigFromFile := config.RouterDefinition
	err := BuildRouter(config.GetRouterType())
	if err != nil {
		openlogging.Error("can not new router: " + err.Error())
		return err
	}

	if routerConfigFromFile != nil {
		if routerConfigFromFile.Destinations != nil {
			DefaultRouter.SetRouteRule(routerConfigFromFile.Destinations)
		}
		if routerConfigFromFile.SourceTemplates != nil {
			Templates = routerConfigFromFile.SourceTemplates
		}
	}

	op, err := getSpecifiedOptions()
	if err != nil {
		return fmt.Errorf("router options error: %v", err)
	}
	DefaultRouter.Init(op)
	openlogging.Info("Router init success")
	return nil
}

// ValidateRule validate the route rules of each service
func ValidateRule(rules map[string][]*model.RouteRule) bool {
	for name, rule := range rules {
		for _, route := range rule {
			allWeight := 0
			for _, routeTag := range route.Routes {
				routeTag.Label = utiltags.LabelOfTags(routeTag.Tags)
				allWeight += routeTag.Weight
			}

			if allWeight > 100 {
				lager.Logger.Warnf("route rule for [%s] is not valid: ruleTag weight is over 100%", name)
				return false
			}
		}

	}
	return true
}

// Options defines how to init router and its fetcher
type Options struct {
	Endpoints []string
	EnableSSL bool
	TLSConfig *tls.Config
	Version   string

	//TODO: need timeout for client
	// TimeOut time.Duration
}

func getSpecifiedOptions() (opts Options, err error) {
	hosts, scheme, err := iputil.URIs2Hosts(strings.Split(config.GetRouterEndpoints(), ","))
	if err != nil {
		return
	}
	opts.Endpoints = hosts
	// TODO: envoy api v1 or v2
	// opts.Version = config.GetRouterAPIVersion()
	opts.TLSConfig, err = chassisTLS.GetTLSConfig(scheme, RouterTLS)
	if err != nil {
		return
	}
	if opts.TLSConfig != nil {
		opts.EnableSSL = true
	}
	return
}

// routeTagToTags returns tags from a route tag
func routeTagToTags(t *model.RouteTag) utiltags.Tags {
	tag := utiltags.Tags{}
	if t != nil {
		tag.KV = make(map[string]string, len(t.Tags))
		for k, v := range t.Tags {
			tag.KV[k] = v
		}
		tag.Label = t.Label
		return tag
	}
	return tag
}
