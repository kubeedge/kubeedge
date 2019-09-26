package log_test

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/common/log"
)

func TestLoggerInitilization(t *testing.T) {
	log.LOGGER.Debug("testing log")
	log.Debugf("testing %s", "beehive")
}
