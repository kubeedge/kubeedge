package config_test

import (
	"fmt"
	"testing"

	"github.com/kubeedge/beehive/pkg/common/config"
)

func TestGetCurrentDirectory(t *testing.T) {
	fmt.Printf(config.CONFIG.GetConfigurationByKey("loggerLevel").(string))
	isEnabled := isModuleEnabled("module_1")
	if !isEnabled {
		t.Error("Error to get modules enabled information")
	}
}

func isModuleEnabled(m string) bool {
	modules := config.CONFIG.GetConfigurationByKey("modules.enabled")
	for _, value := range modules.([]interface{}) {
		if m == value.(string) {
			return true
		}
	}
	return false
}
