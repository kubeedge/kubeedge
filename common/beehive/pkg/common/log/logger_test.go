package log_test

import (
	"testing"

	"github.com/kubeedge/kubeedge/common/beehive/pkg/common/log"
	//"os"
)

func TestLoggerInitilization(t *testing.T) {
	log.LOGGER.Debug("testing log")
	//_, err := os.Stat("edge.log")
	//if err != nil {
	//	t.Error("error when reading log file")
	//}
	//if os.IsNotExist(err) {
	//	t.Error("log file doesn't exist")
	//}
	//err = os.Remove("edge.log")
	//if err != nil {
	//	t.Error("error when reading log file")
	//}
}
