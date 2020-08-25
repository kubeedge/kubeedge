package config

import "strings"

// constant for hystrix keys
const (
	FixedPrefix = "cse"

	NamespaceIsolation                = "isolation"
	NamespaceCircuitBreaker           = "circuitBreaker"
	NamespaceFallback                 = "fallback" //降级
	NamespaceFallbackpolicy           = "fallbackpolicy"
	PropertyTimeoutInMilliseconds     = "timeoutInMilliseconds"
	PropertyTimeoutEnabled            = "timeout.enabled"
	PropertyMaxConcurrentRequests     = "maxConcurrentRequests"
	PropertyErrorThresholdPercentage  = "errorThresholdPercentage"  //失败率
	PropertyRequestVolumeThreshold    = "requestVolumeThreshold"    //窗口请求数
	PropertySleepWindowInMilliseconds = "sleepWindowInMilliseconds" //熔断时间窗
	PropertyEnabled                   = "enabled"
	PropertyForce                     = "force"
	PropertyPolicy                    = "policy"
	PropertyForceClosed               = "forceClosed"
	PropertyForceOpen                 = "forceOpen"
	PropertyFault                     = "fault"
	PropertyGlobal                    = "_global"
	PropertyGovernance                = "governance"
	PropertyConsumer                  = "Consumer"
	PropertySchema                    = "schemas"
	PropertyOperations                = "operations"
	PropertyProtocol                  = "protocols"
	PropertyAbort                     = "abort"
	PropertyPercent                   = "percent"
	PropertyFixedDelay                = "fixedDelay"
	PropertyDelay                     = "delay"
	PropertyHTTPStatus                = "httpStatus"

	LoadBalance = "loadbalance"
)

/*
Hystrix Keys
*/

// GetHystrixSpecificKey get hystrix specific key
func GetHystrixSpecificKey(namespace, cmd, property string) string {
	return strings.Join([]string{FixedPrefix, namespace, cmd, property}, ".")
}

// GetForceFallbackKey get force fallback key
func GetForceFallbackKey(command string) string {
	return GetHystrixSpecificKey(NamespaceFallback, command, PropertyForce)
}

// GetDefaultForceFallbackKey get default force fallback key
func GetDefaultForceFallbackKey(t string) string {
	return GetHystrixSpecificKey(NamespaceFallback, t, PropertyForce)
}

// GetTimeoutKey get timeout key
func GetTimeoutKey(command string) string {
	return GetHystrixSpecificKey(NamespaceIsolation, command, PropertyTimeoutInMilliseconds)
}

// GetDefaultTimeoutKey get default timeout key
func GetDefaultTimeoutKey(t string) string {
	return GetHystrixSpecificKey(NamespaceIsolation, t, PropertyTimeoutInMilliseconds)
}

// GetMaxConcurrentKey get maximum concurrent key
func GetMaxConcurrentKey(command string) string {
	return GetHystrixSpecificKey(NamespaceIsolation, command, PropertyMaxConcurrentRequests)
}

// GetDefaultMaxConcurrentKey get default maximum concurrent key
func GetDefaultMaxConcurrentKey(t string) string {
	return GetHystrixSpecificKey(NamespaceIsolation, t, PropertyMaxConcurrentRequests)
}

// GetErrorPercentThresholdKey get error percentage threshold key
func GetErrorPercentThresholdKey(command string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, command, PropertyErrorThresholdPercentage)
}

// GetDefaultErrorPercentThreshold get default error percentage threshold value
func GetDefaultErrorPercentThreshold(t string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, t, PropertyErrorThresholdPercentage)
}

// GetRequestVolumeThresholdKey get request volume threshold key
func GetRequestVolumeThresholdKey(command string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, command, PropertyRequestVolumeThreshold)
}

// GetDefaultRequestVolumeThresholdKey get default request volume threshold key
func GetDefaultRequestVolumeThresholdKey(t string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, t, PropertyRequestVolumeThreshold)
}

// GetSleepWindowKey get sleep window key
func GetSleepWindowKey(command string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, command, PropertySleepWindowInMilliseconds)
}

// GetDefaultSleepWindowKey get default sleep window key
func GetDefaultSleepWindowKey(t string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, t, PropertySleepWindowInMilliseconds)
}

// GetForceCloseKey get force close key
func GetForceCloseKey(command string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, command, PropertyForceClosed)
}

// GetDefaultForceCloseKey get default force close key
func GetDefaultForceCloseKey(t string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, t, PropertyForceClosed)
}

// GetForceOpenKey get force open key
func GetForceOpenKey(command string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, command, PropertyForceOpen)
}

// GetDefaultForceOpenKey get default force open key
func GetDefaultForceOpenKey(t string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, t, PropertyForceOpen)
}

// GetCircuitBreakerEnabledKey get circuit breaker enabled key
func GetCircuitBreakerEnabledKey(command string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, command, PropertyEnabled)
}

// GetDefaultCircuitBreakerEnabledKey get default circuit breaker enabled key
func GetDefaultCircuitBreakerEnabledKey(t string) string {
	return GetHystrixSpecificKey(NamespaceCircuitBreaker, t, PropertyEnabled)
}

// GetFallbackEnabledKey get fallback enabled key
func GetFallbackEnabledKey(command string) string {
	return GetHystrixSpecificKey(NamespaceFallback, command, PropertyEnabled)
}

// GetDefaultGetFallbackEnabledKey get default fallback enabled key
func GetDefaultGetFallbackEnabledKey(t string) string {
	return GetHystrixSpecificKey(NamespaceFallback, t, PropertyEnabled)
}

// GetFallbackPolicyKey get fallback policy key
func GetFallbackPolicyKey(command string) string {
	return GetHystrixSpecificKey(NamespaceFallbackpolicy, command, PropertyPolicy)
}

// GetDefaultFallbackPolicyKey get default fallback policy key
func GetDefaultFallbackPolicyKey(t string) string {
	return GetHystrixSpecificKey(NamespaceFallbackpolicy, t, PropertyPolicy)
}

// GetFilterNamesKey get filer name and key
func GetFilterNamesKey() string {
	return strings.Join([]string{FixedPrefix, LoadBalance, "serverListFilters"}, ".")
}

// GetFaultInjectionOperationKey get fault injection operation key
func GetFaultInjectionOperationKey(microServiceName, schema, operation string) string {
	return strings.Join([]string{FixedPrefix, PropertyGovernance, PropertyConsumer, microServiceName,
		PropertySchema, schema, PropertyOperations, operation, PropertyPolicy, PropertyFault}, ".")
}

// GetFaultInjectionSchemaKey get fault injection schema key
func GetFaultInjectionSchemaKey(microServiceName, schema string) string {
	return strings.Join([]string{FixedPrefix, PropertyGovernance, PropertyConsumer, microServiceName,
		PropertySchema, schema, PropertyPolicy, PropertyFault}, ".")
}

// GetFaultInjectionServiceKey get fault injection service key
func GetFaultInjectionServiceKey(microServiceName string) string {
	return strings.Join([]string{FixedPrefix, PropertyGovernance, PropertyConsumer, microServiceName, PropertyPolicy, PropertyFault}, ".")
}

// GetFaultInjectionGlobalKey get fault injection global key
func GetFaultInjectionGlobalKey() string {
	return strings.Join([]string{FixedPrefix, PropertyGovernance, PropertyConsumer, PropertyGlobal, PropertyPolicy, PropertyFault}, ".")
}

// GetFaultAbortPercentKey get fault abort percentage key
func GetFaultAbortPercentKey(key, protocol string) string {
	return strings.Join([]string{key, PropertyProtocol, protocol, PropertyAbort, PropertyPercent}, ".")
}

// GetFaultAbortHTTPStatusKey get fault abort http status key
func GetFaultAbortHTTPStatusKey(key, protocol string) string {
	return strings.Join([]string{key, PropertyProtocol, protocol, PropertyAbort, PropertyHTTPStatus}, ".")
}

// GetFaultDelayPercentKey get fault daley percentage key
func GetFaultDelayPercentKey(key, protocol string) string {
	return strings.Join([]string{key, PropertyProtocol, protocol, PropertyDelay, PropertyPercent}, ".")
}

// GetFaultFixedDelayKey get fault fixed delay key
func GetFaultFixedDelayKey(key, protocol string) string {
	return strings.Join([]string{key, PropertyProtocol, protocol, PropertyDelay, PropertyFixedDelay}, ".")
}
