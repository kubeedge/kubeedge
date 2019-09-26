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
		fileutil.GlobalDefinition(),
		fileutil.GetMicroserviceDesc(),
	}
	optionalFiles := []string{
		fileutil.HystrixDefinition(),
		fileutil.GetLoadBalancing(),
		fileutil.GetRateLimiting(),
		fileutil.GetTLS(),
		fileutil.GetMonitoring(),
		fileutil.GetAuth(),
		fileutil.GetTracing(),
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
