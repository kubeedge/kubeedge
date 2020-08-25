package config

import (
	"github.com/go-chassis/go-archaius"

	"strconv"
	"time"
)

// constant for default values of abort and delay
const (
	DefaultAbortPercent = 0
	DefaultAbortStatus  = 0
	DefaultDelayPercent = 0
)

// GetAbortPercent get abort percentage
func GetAbortPercent(protocol, microServiceName, schema, operation string) int {

	var key string
	var abortPercent int
	if microServiceName != "" && schema != "" && operation != "" {
		key = GetFaultInjectionOperationKey(microServiceName, schema, operation)
		abortPercent = archaius.GetInt(GetFaultAbortPercentKey(key, protocol), DefaultAbortPercent)
	}
	if abortPercent == 0 && microServiceName != "" && schema != "" {
		key = GetFaultInjectionSchemaKey(microServiceName, schema)
		abortPercent = archaius.GetInt(GetFaultAbortPercentKey(key, protocol), DefaultAbortPercent)
	}
	if abortPercent == 0 && microServiceName != "" {
		key = GetFaultInjectionServiceKey(microServiceName)
		abortPercent = archaius.GetInt(GetFaultAbortPercentKey(key, protocol), DefaultAbortPercent)
	}
	if abortPercent == 0 {
		key = GetFaultInjectionGlobalKey()
		abortPercent = archaius.GetInt(GetFaultAbortPercentKey(key, protocol), DefaultAbortPercent)
	}

	return abortPercent
}

// GetAbortStatus get abort status
func GetAbortStatus(protocol, microServiceName, schema, operation string) int {

	var key string
	var abortHTTPStatus int
	if microServiceName != "" && schema != "" && operation != "" {
		key = GetFaultInjectionOperationKey(microServiceName, schema, operation)
		abortHTTPStatus = archaius.GetInt(GetFaultAbortHTTPStatusKey(key, protocol), DefaultAbortStatus)
	}
	if abortHTTPStatus == 0 && microServiceName != "" && schema != "" {
		key = GetFaultInjectionSchemaKey(microServiceName, schema)
		abortHTTPStatus = archaius.GetInt(GetFaultAbortHTTPStatusKey(key, protocol), DefaultAbortStatus)
	}
	if abortHTTPStatus == 0 && microServiceName != "" {
		key = GetFaultInjectionServiceKey(microServiceName)
		abortHTTPStatus = archaius.GetInt(GetFaultAbortHTTPStatusKey(key, protocol), DefaultAbortStatus)
	}
	if abortHTTPStatus == 0 {
		key = GetFaultInjectionGlobalKey()
		abortHTTPStatus = archaius.GetInt(GetFaultAbortHTTPStatusKey(key, protocol), DefaultAbortStatus)
	}

	return abortHTTPStatus
}

// GetDelayPercent get delay percentage
func GetDelayPercent(protocol, microServiceName, schema, operation string) int {

	var key string
	var delayPercent int
	if microServiceName != "" && schema != "" && operation != "" {
		key = GetFaultInjectionOperationKey(microServiceName, schema, operation)
		delayPercent = archaius.GetInt(GetFaultDelayPercentKey(key, protocol), DefaultDelayPercent)
	}
	if delayPercent == 0 && microServiceName != "" && schema != "" {
		key = GetFaultInjectionSchemaKey(microServiceName, schema)
		delayPercent = archaius.GetInt(GetFaultDelayPercentKey(key, protocol), DefaultDelayPercent)
	}
	if delayPercent == 0 && microServiceName != "" {
		key = GetFaultInjectionServiceKey(microServiceName)
		delayPercent = archaius.GetInt(GetFaultDelayPercentKey(key, protocol), DefaultDelayPercent)
	}
	if delayPercent == 0 {
		key = GetFaultInjectionGlobalKey()
		delayPercent = archaius.GetInt(GetFaultDelayPercentKey(key, protocol), DefaultDelayPercent)
	}

	return delayPercent
}

// GetFixedDelay get fixed delay
func GetFixedDelay(protocol, microServiceName, schema, operation string) time.Duration {

	var key string
	var fixedDelayTime time.Duration
	var fixedDelay interface{}
	if microServiceName != "" && schema != "" && operation != "" {
		key = GetFaultInjectionOperationKey(microServiceName, schema, operation)
		fixedDelay = archaius.Get(GetFaultFixedDelayKey(key, protocol))
	}
	if fixedDelay == nil && microServiceName != "" && schema != "" {
		key = GetFaultInjectionSchemaKey(microServiceName, schema)
		fixedDelay = archaius.Get(GetFaultFixedDelayKey(key, protocol))
	}
	if fixedDelay == nil && microServiceName != "" {
		key = GetFaultInjectionServiceKey(microServiceName)
		fixedDelay = archaius.Get(GetFaultFixedDelayKey(key, protocol))
	}
	if fixedDelay == nil {
		key = GetFaultInjectionGlobalKey()
		fixedDelay = archaius.Get(GetFaultFixedDelayKey(key, protocol))
	}
	switch fixedDelay.(type) {
	case int:
		fixedDelayInt := fixedDelay.(int)
		fixedDelayTime = time.Duration(fixedDelayInt) * time.Millisecond
	case string:
		fixedDelayInt, _ := strconv.Atoi(fixedDelay.(string))
		fixedDelayTime = time.Duration(fixedDelayInt) * time.Millisecond
	}
	return fixedDelayTime
}
