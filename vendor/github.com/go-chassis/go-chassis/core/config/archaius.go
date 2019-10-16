package config

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/pkg/util/fileutil"
	"time"
)

// InitArchaius initialize the archaius
func InitArchaius() error {
	var err error

	requiredFiles := []string{
		fileutil.GlobalConfigPath(),
		fileutil.MicroServiceConfigPath(),
	}
	optionalFiles := []string{
		fileutil.CircuitBreakerConfigPath(),
		fileutil.LoadBalancingConfigPath(),
		fileutil.RateLimitingFile(),
		fileutil.TLSConfigPath(),
		fileutil.MonitoringConfigPath(),
		fileutil.AuthConfigPath(),
		fileutil.TracingPath(),
		fileutil.LogConfigPath(),
		fileutil.RouterConfigPath(),
	}

	err = archaius.Init(
		archaius.WithCommandLineSource(),
		archaius.WithMemorySource(),
		archaius.WithENVSource(),
		archaius.WithRequiredFiles(requiredFiles),
		archaius.WithOptionalFiles(optionalFiles))

	return err
}

// GetTimeoutDurationFromArchaius get timeout durations from archaius
func GetTimeoutDurationFromArchaius(service, t string) time.Duration {
	timeout := archaius.GetInt(GetTimeoutKey(service), archaius.GetInt(GetDefaultTimeoutKey(t), DefaultTimeout))
	return time.Duration(timeout) * time.Millisecond
}
