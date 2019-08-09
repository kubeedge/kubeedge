// Package loadbalancer is client side load balancer
package loadbalancer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/registry"
	"github.com/go-chassis/go-chassis/pkg/util/tags"
	"github.com/go-mesh/openlogging"
)

// constant string for zoneaware
const (
	ZoneAware = "zoneaware"
)

//StrategyLatency is name
const StrategyLatency = "WeightedResponse"

// constant strings for load balance variables
const (
	StrategyRoundRobin        = "RoundRobin"
	StrategyRandom            = "Random"
	StrategySessionStickiness = "SessionStickiness"

	OperatorEqual   = "="
	OperatorGreater = ">"
	OperatorSmaller = "<"
	OperatorPattern = "Pattern"
)

var (
	// ErrNoneAvailableInstance is to represent load balance error
	ErrNoneAvailableInstance = LBError{Message: "None available instance"}
)

// LBError load balance error
type LBError struct {
	Message string
}

// Error for to return load balance error message
func (e LBError) Error() string {
	return "lb: " + e.Message
}

// BuildStrategy query instance list and give it to Strategy then return Strategy
func BuildStrategy(i *invocation.Invocation,
	s Strategy) (Strategy, error) {

	if s == nil {
		s = &RoundRobinStrategy{}
	}

	var isFilterExist = true
	for _, filter := range i.Filters {
		if filter == "" {
			isFilterExist = false
		}

	}

	instances, err := registry.DefaultServiceDiscoveryService.FindMicroServiceInstances(i.SourceServiceID, i.MicroServiceName, i.RouteTags)
	if err != nil {
		lbErr := LBError{err.Error()}
		openlogging.GetLogger().Errorf("Lb err: %s", err)
		return nil, lbErr
	}

	if isFilterExist {
		filterFuncs := make([]Filter, 0)
		//append filters in config
		for _, fName := range i.Filters {
			f := Filters[fName]
			if f != nil {
				filterFuncs = append(filterFuncs, f)
				continue
			}
		}
		for _, filter := range filterFuncs {
			instances = filter(instances, nil)
		}
	}

	if len(instances) == 0 {
		lbErr := LBError{fmt.Sprintf("No available instance, key: %s(%v)", i.MicroServiceName, i.RouteTags)}
		openlogging.Error(lbErr.Error())
		return nil, lbErr
	}

	serviceKey := strings.Join([]string{i.MicroServiceName, i.RouteTags.String()}, "|")
	s.ReceiveData(i, instances, serviceKey)
	return s, nil
}

// Strategy is load balancer algorithm , call Pick to return one instance
type Strategy interface {
	ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceKey string)
	Pick() (*registry.MicroServiceInstance, error)
}

//Criteria is rule for filter
type Criteria struct {
	Key      string
	Operator string
	Value    string
}

// Filter receive instances and criteria, it will filter instances based on criteria you defined,criteria is optional, you can give nil for it
type Filter func(instances []*registry.MicroServiceInstance, criteria []*Criteria) []*registry.MicroServiceInstance

// Enable function is for to enable load balance strategy
func Enable(strategyName string) error {
	openlogging.Info("Enable LoadBalancing")
	InstallStrategy(StrategyRandom, newRandomStrategy)
	InstallStrategy(StrategyRoundRobin, newRoundRobinStrategy)
	InstallStrategy(StrategySessionStickiness, newSessionStickinessStrategy)

	if strategyName == "" {
		openlogging.Info("Empty strategy configuration, use RoundRobin as default")
		return nil
	}
	openlogging.Info("Strategy is " + strategyName)

	return nil
}

// Filters is a map of string and array of *registry.MicroServiceInstance
var Filters = make(map[string]Filter)

// InstallFilter install filter
func InstallFilter(name string, f Filter) {
	Filters[name] = f
}

// variables for latency map, rest and highway requests count
var (
	//ProtocolStatsMap saves all stats for all service's protocol, one protocol has a lot of instances
	ProtocolStatsMap = make(map[string][]*ProtocolStats)
	//maintain different locks since multiple goroutine access the map
	LatencyMapRWMutex sync.RWMutex
)

//BuildKey return key of stats map
func BuildKey(microServiceName, tags, protocol string) string {
	//TODO add more data
	return strings.Join([]string{microServiceName, tags, protocol}, "/")
}

// SetLatency for a instance ,it only save latest 10 stats for instance's protocol
func SetLatency(latency time.Duration, addr, microServiceName string, tags utiltags.Tags, protocol string) {
	key := BuildKey(microServiceName, tags.String(), protocol)

	LatencyMapRWMutex.RLock()
	stats, ok := ProtocolStatsMap[key]
	LatencyMapRWMutex.RUnlock()
	if !ok {
		stats = make([]*ProtocolStats, 0)
	}
	exist := false
	for _, v := range stats {
		if v.Addr == addr {
			v.SaveLatency(latency)
			exist = true
		}
	}
	if !exist {
		ps := &ProtocolStats{
			Addr: addr,
		}

		ps.SaveLatency(latency)
		stats = append(stats, ps)
	}
	LatencyMapRWMutex.Lock()
	ProtocolStatsMap[key] = stats
	LatencyMapRWMutex.Unlock()
}
