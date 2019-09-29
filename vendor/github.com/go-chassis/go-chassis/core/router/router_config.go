package router

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-chassis/go-chassis/core/config"
	chassisTLS "github.com/go-chassis/go-chassis/core/tls"
	"github.com/go-chassis/go-chassis/pkg/util/iputil"
	"github.com/go-chassis/go-chassis/pkg/util/tags"
	"github.com/go-mesh/openlogging"
)

// RouterTLS defines tls prefix
const RouterTLS = "router"

//Init initialize router config in local file
//then is create the router component
func Init() error {
	OldRouteRule := config.OldRouterDefinition // compatible with old local configs, it is read from old config format
	err := BuildRouter(config.GetRouterType())
	if err != nil {
		openlogging.Error("can not init router [" + config.GetRouterType() + "]: " + err.Error())
		return err
	}
	if OldRouteRule != nil {
		if OldRouteRule.SourceTemplates != nil {
			Templates = OldRouteRule.SourceTemplates
		}
	}

	op, err := getSpecifiedOptions()
	if err != nil {
		return fmt.Errorf("router options error: %v", err)
	}
	err = DefaultRouter.Init(op)
	if err != nil {
		openlogging.Error(err.Error())
		return err
	}
	openlogging.Info("router init success")
	return nil
}

// ValidateRule validate the route rules of each service
func ValidateRule(rules map[string][]*config.RouteRule) bool {
	for name, rule := range rules {
		for _, route := range rule {
			allWeight := 0
			for _, routeTag := range route.Routes {
				routeTag.Label = utiltags.LabelOfTags(routeTag.Tags)
				allWeight += routeTag.Weight
			}

			if allWeight > 100 {
				openlogging.Warn("route rule is invalid: total weight is over 100%", openlogging.WithTags(
					openlogging.Tags{
						"service": name,
					}))
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
func routeTagToTags(t *config.RouteTag) utiltags.Tags {
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
