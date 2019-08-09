// Package router expose API for user to get or set route rule
package router

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/registry"
	wp "github.com/go-chassis/go-chassis/core/router/weightpool"
	"github.com/go-mesh/openlogging"
)

//Templates is for source match template settings
var Templates = make(map[string]*model.Match)

//Router return route rule, you can also set custom route rule
type Router interface {
	Init(Options) error
	SetRouteRule(map[string][]*model.RouteRule)
	FetchRouteRuleByServiceName(service string) []*model.RouteRule
}

// ErrNoExist means if there is no router implementation
var ErrNoExist = errors.New("router not exists")
var routerServices = make(map[string]func() (Router, error))

// DefaultRouter is current router implementation
var DefaultRouter Router

// InstallRouterService install router service for developer
func InstallRouterService(name string, f func() (Router, error)) {
	openlogging.Info("install route rule plugin: " + name)
	routerServices[name] = f
}

//BuildRouter create a router
func BuildRouter(name string) error {
	f, ok := routerServices[name]
	if !ok {
		return ErrNoExist
	}
	r, err := f()
	if err != nil {
		return err
	}
	DefaultRouter = r
	return nil
}

//Route decide the target service metadata
//it decide based on configuration of route rule
//it will set RouteTag to invocation
func Route(header map[string]string, si *registry.SourceInfo, inv *invocation.Invocation) error {
	rules := SortRules(inv.MicroServiceName)
	for _, rule := range rules {
		if Match(rule.Match, header, si) {
			tag := FitRate(rule.Routes, inv.MicroServiceName)
			inv.RouteTags = routeTagToTags(tag)
			break
		}
	}
	return nil
}

// FitRate fit rate
func FitRate(tags []*model.RouteTag, dest string) *model.RouteTag {
	if tags[0].Weight == 100 {
		return tags[0]
	}

	pool, ok := wp.GetPool().Get(dest)
	if !ok {
		// first request route to tags[0]
		wp.GetPool().Set(dest, wp.NewPool(tags...))
		return tags[0]
	}
	return pool.PickOne()
}

// Match check the route rule
func Match(match model.Match, headers map[string]string, source *registry.SourceInfo) bool {
	//validate template first
	if refer := match.Refer; refer != "" {
		return SourceMatch(Templates[refer], headers, source)
	}
	//match rule is not set
	if match.Source == "" && match.HTTPHeaders == nil && match.Headers == nil {
		return true
	}

	return SourceMatch(&match, headers, source)
}

// SourceMatch check the source route
func SourceMatch(match *model.Match, headers map[string]string, source *registry.SourceInfo) bool {
	//source not match
	if match.Source != "" && match.Source != source.Name {
		return false
	}
	//source tags not match
	if len(match.SourceTags) != 0 {
		for k, v := range match.SourceTags {
			if v != source.Tags[k] {
				return false
			}
		}
	}

	//source headers not match
	if match.Headers != nil {
		for k, v := range match.Headers {
			if !isMatch(headers, k, v) {
				return false
			}
			continue
		}
	}
	if match.HTTPHeaders != nil {
		for k, v := range match.HTTPHeaders {
			if !isMatch(headers, k, v) {
				return false
			}
			continue
		}
	}
	return true
}

// isMatch check the route rule
func isMatch(headers map[string]string, k string, v map[string]string) bool {
	header := valueToUpper(v["caseInsensitive"], headers[k])

	if regex, ok := v["regex"]; ok {

		reg := regexp.MustCompilePOSIX(valueToUpper(v["caseInsensitive"], regex))
		if !reg.Match([]byte(header)) {
			return false
		}
		return true

	}
	if exact, ok := v["exact"]; ok {
		if valueToUpper(v["caseInsensitive"], exact) != header {
			return false
		}
		return true
	}
	if noEqu, ok := v["noEqu"]; ok {
		if valueToUpper(v["caseInsensitive"], noEqu) == header {
			return false
		}
		return true
	}

	headerInt, err := strconv.Atoi(header)
	if err != nil {
		return false
	}
	if noLess, ok := v["noLess"]; ok {
		head, _ := strconv.Atoi(noLess)
		if head > headerInt {
			return false
		}
		return true
	}
	if noGreater, ok := v["noGreater"]; ok {
		head, _ := strconv.Atoi(noGreater)
		if head < headerInt {
			return false
		}
		return true
	}
	if greater, ok := v["greater"]; ok {
		head, _ := strconv.Atoi(greater)
		if head >= headerInt {
			return false
		}
		return true
	}
	if less, ok := v["less"]; ok {
		head, _ := strconv.Atoi(less)
		if head <= headerInt {
			return false
		}
	}
	return true
}
func valueToUpper(b, value string) string {
	if b == common.TRUE {
		value = strings.ToUpper(value)
	}

	return value
}

// SortRules sort route rules
func SortRules(name string) []*model.RouteRule {
	slice := DefaultRouter.FetchRouteRuleByServiceName(name)
	return QuickSort(0, len(slice)-1, slice)
}

// QuickSort for sorting the routes it will follow quicksort technique
func QuickSort(left int, right int, rules []*model.RouteRule) (s []*model.RouteRule) {
	s = rules
	if left >= right {
		return
	}

	i := left
	j := right
	base := s[left]
	var tmp *model.RouteRule
	for i != j {
		for s[j].Precedence <= base.Precedence && i < j {
			j--
		}
		for s[i].Precedence >= base.Precedence && i < j {
			i++
		}
		if i < j {
			tmp = s[i]
			s[i] = s[j]
			s[j] = tmp
		}
	}
	//move base to the current position of i&j
	s[left] = s[i]
	s[i] = base

	QuickSort(left, i-1, s)
	QuickSort(i+1, right, s)

	return
}
