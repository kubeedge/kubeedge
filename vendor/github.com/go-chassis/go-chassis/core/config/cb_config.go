package config

import (
	"sync"

	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config/model"
	"time"
)

// constant for hystrix parameters
const (
	DefaultForceFallback                 = false
	DefaultTimeoutEnabled                = false
	DefaultConsumerCircuitBreakerEnabled = false
	DefaultProviderCircuitBreakerEnabled = false
	DefaultCircuitBreakerForceOpen       = false
	DefaultCircuitBreakerForceClosed     = false
	DefaultFallbackEnable                = true
	DefaultMaxConcurrent                 = 1000
	DefaultSleepWindow                   = 15000
	DefaultTimeout                       = 30000
	DefaultErrorPercentThreshold         = 50
	DefaultRequestVolumeThreshold        = 20
	PolicyNull                           = "returnnull"
	PolicyThrowException                 = "throwexception"
)

var cbMutex = sync.RWMutex{}

// GetFallbackEnabled get fallback enabled
func GetFallbackEnabled(command, t string) bool {
	return archaius.GetBool(GetFallbackEnabledKey(command),
		archaius.GetBool(GetDefaultGetFallbackEnabledKey(t), DefaultFallbackEnable))
}

// GetCircuitBreakerEnabled get circuit breaker enabled
func GetCircuitBreakerEnabled(command, t string) bool {
	if common.Consumer == command {
		return archaius.GetBool(GetCircuitBreakerEnabledKey(command),
			archaius.GetBool(GetDefaultCircuitBreakerEnabledKey(t), DefaultConsumerCircuitBreakerEnabled))
	}

	return archaius.GetBool(GetCircuitBreakerEnabledKey(command),
		archaius.GetBool(GetDefaultCircuitBreakerEnabledKey(t), DefaultProviderCircuitBreakerEnabled))
}

// GetForceClose get force close
func GetForceClose(service, t string) bool {
	cbMutex.RLock()
	cbspec := getCircuitBreakerSpec(t)
	if cb, ok := cbspec.AnyService[service]; ok {
		cbMutex.RUnlock()
		return cb.ForceClose
	}
	cbMutex.RUnlock()
	return cbspec.ForceClose
}

// GetForceOpen get foce open
func GetForceOpen(service, t string) bool {
	cbMutex.RLock()
	cbspec := getCircuitBreakerSpec(t)
	if cb, ok := cbspec.AnyService[service]; ok {
		cbMutex.RUnlock()
		return cb.ForceOpen
	}
	cbMutex.RUnlock()
	return cbspec.ForceOpen
}

// GetTimeout get timeout durations
func GetTimeout(service, t string) int {
	cbMutex.RLock()
	global := getIsolationSpec(t).TimeoutInMilliseconds
	if global == 0 {
		global = DefaultTimeout
	}
	m := archaius.GetInt(GetTimeoutKey(service), global)
	cbMutex.RUnlock()
	return m
}

// GetTimeoutDuration get timeout durations from cache first, then get from archaius
func GetTimeoutDuration(service, t string) time.Duration {
	timeout := GetTimeout(service, t)
	return time.Duration(timeout) * time.Millisecond
}

// GetMaxConcurrentRequests get max concurrent requests
func GetMaxConcurrentRequests(command, t string) int {
	cbMutex.RLock()
	global := getIsolationSpec(t).MaxConcurrentRequests
	if global == 0 {
		global = DefaultMaxConcurrent
	}
	m := archaius.GetInt(GetMaxConcurrentKey(command), global)
	cbMutex.RUnlock()
	return m
}

// GetErrorPercentThreshold get error percent threshold
func GetErrorPercentThreshold(command, t string) int {
	cbMutex.RLock()
	global := getCircuitBreakerSpec(t).ErrorThresholdPercentage
	if global == 0 {
		global = DefaultErrorPercentThreshold
	}
	m := archaius.GetInt(GetErrorPercentThresholdKey(command), global)
	cbMutex.RUnlock()
	return m
}

// GetRequestVolumeThreshold get request volume threshold
func GetRequestVolumeThreshold(command, t string) int {
	cbMutex.RLock()
	global := getCircuitBreakerSpec(t).RequestVolumeThreshold
	if global == 0 {
		global = DefaultRequestVolumeThreshold
	}
	m := archaius.GetInt(GetRequestVolumeThresholdKey(command), global)
	cbMutex.RUnlock()
	return m
}

// GetSleepWindow get sleep window
func GetSleepWindow(command, t string) int {
	cbMutex.RLock()
	global := getCircuitBreakerSpec(t).SleepWindowInMilliseconds
	if global == 0 {
		global = DefaultSleepWindow
	}
	m := archaius.GetInt(GetSleepWindowKey(command), global)
	cbMutex.RUnlock()
	return m
}

// GetPolicy get fallback policy
func GetPolicy(service, t string) string {
	cbMutex.RLock()
	policy := getFallbackPolicySpec(t).AnyService[service].Policy
	if policy == "" {
		policy = getFallbackPolicySpec(t).Policy
		if policy == "" {
			policy = PolicyThrowException
		}
	}
	cbMutex.RUnlock()
	return policy
}

func getIsolationSpec(command string) *model.IsolationSpec {
	if command == common.Consumer {
		return GetHystrixConfig().IsolationProperties.Consumer
	}
	return GetHystrixConfig().IsolationProperties.Provider
}

func getCircuitBreakerSpec(command string) *model.CircuitBreakerSpec {
	if command == common.Consumer {
		return GetHystrixConfig().CircuitBreakerProperties.Consumer
	}
	return GetHystrixConfig().CircuitBreakerProperties.Provider
}

func getFallbackSpec(command string) *model.FallbackSpec {
	if command == common.Consumer {
		return GetHystrixConfig().FallbackProperties.Consumer
	}
	return GetHystrixConfig().FallbackProperties.Provider
}

func getFallbackPolicySpec(command string) *model.FallbackPolicySpec {
	if command == common.Consumer {
		return GetHystrixConfig().FallbackPolicyProperties.Consumer
	}
	return GetHystrixConfig().FallbackPolicyProperties.Provider
}

// GetForceFallback get force fallback
func GetForceFallback(service, t string) bool {
	cbMutex.RLock()
	fallback := getFallbackSpec(t)
	if en, ok := fallback.AnyService[service]; ok {
		cbMutex.RUnlock()
		return en.Force
	}
	cbMutex.RUnlock()
	return fallback.Force
}
