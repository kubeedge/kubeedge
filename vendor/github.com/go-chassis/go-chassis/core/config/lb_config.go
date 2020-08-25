package config

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/pkg/backoff"
	"strings"
	"sync"
)

const (
	lbPrefix                                 = "cse.loadbalance"
	propertyStrategyName                     = "strategy.name"
	propertySessionStickinessRuleTimeout     = "SessionStickinessRule.sessionTimeoutInSeconds"
	propertySessionStickinessRuleFailedTimes = "SessionStickinessRule.successiveFailedTimes"
	propertyRetryEnabled                     = "retryEnabled"
	propertyRetryOnNext                      = "retryOnNext"
	propertyRetryOnSame                      = "retryOnSame"
	propertyBackoffKind                      = "backoff.kind"
	propertyBackoffMinMs                     = "backoff.minMs"
	propertyBackoffMaxMs                     = "backoff.maxMs"

	//DefaultStrategy is default value for strategy
	DefaultStrategy = "RoundRobin"
	//DefaultSessionTimeout is default value for timeout
	DefaultSessionTimeout = 30
	//DefaultFailedTimes is default value for failed times
	DefaultFailedTimes = 5
)

var lbMutex = sync.RWMutex{}

func genKey(s ...string) string {
	return strings.Join(s, ".")
}

// GetServerListFilters get server list filters
func GetServerListFilters() (filters []string) {
	lbMutex.RLock()
	filters = strings.Split(GetLoadBalancing().Filters, ",")
	lbMutex.RUnlock()
	return
}

// GetStrategyName get strategy name
func GetStrategyName(source, service string) string {
	lbMutex.RLock()
	r := GetLoadBalancing().AnyService[service].Strategy["name"]
	if r == "" {
		r = GetLoadBalancing().Strategy["name"]
		if r == "" {
			r = DefaultStrategy
		}
	}
	lbMutex.RUnlock()
	return r
}

// GetSessionTimeout return session timeout
func GetSessionTimeout(source, service string) int {
	lbMutex.RLock()
	global := GetLoadBalancing().SessionStickinessRule.SessionTimeoutInSeconds
	if global == 0 {
		global = DefaultSessionTimeout
	}
	ms := archaius.GetInt(genKey(lbPrefix, service, propertySessionStickinessRuleTimeout), global)
	lbMutex.RUnlock()
	return ms
}

// StrategySuccessiveFailedTimes strategy successive failed times
func StrategySuccessiveFailedTimes(source, service string) int {
	lbMutex.RLock()
	global := GetLoadBalancing().SessionStickinessRule.SuccessiveFailedTimes
	if global == 0 {
		global = DefaultFailedTimes
	}
	ms := archaius.GetInt(genKey(lbPrefix, service, propertySessionStickinessRuleFailedTimes), global)
	lbMutex.RUnlock()
	return ms
}

// RetryEnabled retry enabled
func RetryEnabled(source, service string) bool {
	lbMutex.RLock()
	global := GetLoadBalancing().RetryEnabled
	ms := archaius.GetBool(genKey(lbPrefix, service, propertyRetryEnabled), global)
	lbMutex.RUnlock()
	return ms
}

//GetRetryOnNext return value of GetRetryOnNext
func GetRetryOnNext(source, service string) int {
	lbMutex.RLock()
	global := GetLoadBalancing().RetryOnNext
	ms := archaius.GetInt(genKey(lbPrefix, service, propertyRetryOnNext), global)
	lbMutex.RUnlock()
	return ms
}

//GetRetryOnSame return value of RetryOnSame
func GetRetryOnSame(source, service string) int {
	lbMutex.RLock()
	global := GetLoadBalancing().RetryOnSame
	ms := archaius.GetInt(genKey(lbPrefix, service, propertyRetryOnSame), global)
	lbMutex.RUnlock()
	return ms
}

//BackOffKind get kind
func BackOffKind(source, service string) string {
	r := GetLoadBalancing().AnyService[service].Backoff.Kind
	if r == "" {
		r = GetLoadBalancing().Backoff.Kind
		if r == "" {
			r = backoff.DefaultBackOffKind
		}
	}
	return r
}

//BackOffMinMs get min time
func BackOffMinMs(source, service string) int {
	global := GetLoadBalancing().Backoff.MinMs
	ms := archaius.GetInt(genKey(lbPrefix, service, propertyBackoffMinMs), global)
	return ms
}

//BackOffMaxMs get max time
func BackOffMaxMs(source, service string) int {
	global := GetLoadBalancing().Backoff.MaxMs
	ms := archaius.GetInt(genKey(lbPrefix, service, propertyBackoffMaxMs), global)
	return ms
}
