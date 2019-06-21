package config

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/pkg/util/fileutil"
	"sync"
)

var initArchaiusOnce = sync.Once{}

// InitArchaius initialize the archaius
func InitArchaius() error {
	var err error
	initArchaiusOnce.Do(func() {
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
			fileutil.PaasLagerDefinition(),
		}

		err = archaius.Init(
			archaius.WithCommandLineSource(),
			archaius.WithMemorySource(),
			archaius.WithENVSource(),
			archaius.WithRequiredFiles(requiredFiles),
			archaius.WithOptionalFiles(optionalFiles))
	})

	return err
}
