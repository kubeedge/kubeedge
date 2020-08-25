package model

import (
	"gopkg.in/yaml.v2"
)

// HystrixConfigWrapper hystrix configuration wrapper structure
type HystrixConfigWrapper struct {
	HystrixConfig *HystrixConfig `yaml:"cse"`
}

// HystrixConfig is hystrix configuration structure
type HystrixConfig struct {
	IsolationProperties      *IsolationWrapper      `yaml:"isolation"`
	CircuitBreakerProperties *CircuitWrapper        `yaml:"circuitBreaker"`
	FallbackProperties       *FallbackWrapper       `yaml:"fallback"`
	FallbackPolicyProperties *FallbackPolicyWrapper `yaml:"fallbackpolicy"`
}

// IsolationWrapper isolation wrapper structure
type IsolationWrapper struct {
	Consumer *IsolationSpec `yaml:"Consumer"`
	Provider *IsolationSpec `yaml:"Provider"`
}

// CircuitWrapper circuit wrapper structure
type CircuitWrapper struct {
	Scope    string              `yaml:"scope"`
	Consumer *CircuitBreakerSpec `yaml:"Consumer"`
	Provider *CircuitBreakerSpec `yaml:"Provider"`
}

// FallbackWrapper fallback wrapper structure
type FallbackWrapper struct {
	Consumer *FallbackSpec `yaml:"Consumer"`
	Provider *FallbackSpec `yaml:"Provider"`
}

// FallbackPolicyWrapper fallback policy wrapper
type FallbackPolicyWrapper struct {
	Consumer *FallbackPolicySpec `yaml:"Consumer"`
	Provider *FallbackPolicySpec `yaml:"Provider"`
}

// IsolationSpec isolation speciafications
type IsolationSpec struct {
	TimeoutInMilliseconds int                      `yaml:"timeoutInMilliseconds"`
	MaxConcurrentRequests int                      `yaml:"maxConcurrentRequests"`
	AnyService            map[string]IsolationSpec `yaml:",inline"`
}

// CircuitBreakerSpec circuit breaker specifications
type CircuitBreakerSpec struct {
	Enabled                   bool                                  `yaml:"enabled"`
	ForceOpen                 bool                                  `yaml:"forceOpen"`
	ForceClose                bool                                  `yaml:"forceClosed"`
	SleepWindowInMilliseconds int                                   `yaml:"sleepWindowInMilliseconds"`
	RequestVolumeThreshold    int                                   `yaml:"requestVolumeThreshold"`
	ErrorThresholdPercentage  int                                   `yaml:"errorThresholdPercentage"`
	AnyService                map[string]CircuitBreakPropertyStruct `yaml:",inline"`
}

// FallbackSpec fallback specifications
type FallbackSpec struct {
	Enabled               bool                              `yaml:"enabled"`
	Force                 bool                              `yaml:"force"`
	MaxConcurrentRequests int                               `yaml:"maxConcurrentRequests"`
	AnyService            map[string]FallbackPropertyStruct `yaml:",inline"`
}

// FallbackPolicySpec fallback policy specifications
type FallbackPolicySpec struct {
	Policy     string                                  `yaml:"policy"`
	AnyService map[string]FallbackPolicyPropertyStruct `yaml:",inline"`
}

// IsolationPropertyStruct isolation 属性集合
type IsolationPropertyStruct struct {
	TimeoutInMilliseconds int `yaml:"timeoutInMilliseconds"`
	MaxConcurrentRequests int `yaml:"maxConcurrentRequests"`
}

// CircuitBreakPropertyStruct circuitBreaker 属性集合
type CircuitBreakPropertyStruct struct {
	Enabled                   bool `yaml:"enabled"`
	ForceOpen                 bool `yaml:"forceOpen"`
	ForceClose                bool `yaml:"forceClosed"`
	SleepWindowInMilliseconds int  `yaml:"sleepWindowInMilliseconds"`
	RequestVolumeThreshold    int  `yaml:"requestVolumeThreshold"`
	ErrorThresholdPercentage  int  `yaml:"errorThresholdPercentage"`
}

// FallbackPropertyStruct fallback property structure
type FallbackPropertyStruct struct {
	Enabled               bool `yaml:"enabled"`
	Force                 bool `yaml:"force"`
	MaxConcurrentRequests int  `yaml:"maxConcurrentRequests"`
}

// FallbackPolicyPropertyStruct fallback policy property structure
type FallbackPolicyPropertyStruct struct {
	Policy string `yaml:"policy"`
}

// constant for consumer and provider
const (
	ConsumerType = "Consumer"
	ProviderType = "Provider"
)

// variables of isolation, circuit, fallback
var (
	//default config fo hystric.
	DefaultIsolation = IsolationPropertyStruct{
		TimeoutInMilliseconds: 1000,
		MaxConcurrentRequests: 4000,
	}
	DefaultCircuit = CircuitBreakPropertyStruct{
		Enabled:                   true,
		ForceOpen:                 false,
		ForceClose:                false,
		SleepWindowInMilliseconds: 5000,
		RequestVolumeThreshold:    20,
		ErrorThresholdPercentage:  50,
	}
	DefaultFallback = FallbackPropertyStruct{
		Enabled:               true,
		MaxConcurrentRequests: 4000,
	}
)

// String returns marshalling data of hystrix config wrapper
func (hc *HystrixConfigWrapper) String() ([]byte, error) {
	return yaml.Marshal(hc)
}
