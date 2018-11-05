package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ServiceComb/paas-lager"
	"github.com/ServiceComb/paas-lager/rotate"
	"github.com/go-chassis/paas-lager/third_party/forked/cloudfoundry/lager"
	"gopkg.in/yaml.v2"

	"kubeedge/beehive/pkg/common/config"
)

// LOGGER var
var LOGGER lager.Logger

func init() {

	logConfig := &log.Config{
		LoggerLevel:   "DEBUG",
		EnableRsyslog: false,
		LogFormatText: true,
		Writers:       []string{"stdout"},
	}

	filename := config.GetConfigDirectory() + "/conf/logging.yaml"

	fmt.Printf(filename)
	_, err := os.Stat(filename)
	if err == nil {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("Got error when reading yaml config file:%v\n", err)
		}
		err = yaml.Unmarshal(content, logConfig)
		if err != nil {
			fmt.Printf("Got error when reading yaml config file:%v\n", err)
		}
		fmt.Printf("logConfig:%v\n", logConfig)
	} else {
		fmt.Printf("Got error when reading yaml config file:%v\n", err)
	}

	log.Init(*logConfig)

	LOGGER = log.NewLogger("kubeedge")
	writers := logConfig.Writers
	for _, value := range writers {
		if value == "file" {
			rotate.RunLogRotate(logConfig.LoggerFile, &rotate.RotateConfig{}, LOGGER)
			break
		}
	}

	LOGGER.Debug("init logger...")
}

// Trace 放在一个函数的开头，会打印整个函数的运行时间
func Trace(funcName string, msg ...string) func() {
	start := time.Now()
	s := funcName
	for _, i := range msg {
		s = fmt.Sprintf("%s [%s]", s, i)
	}
	LOGGER.Infof("enter %s ...", s)
	return func() {
		LOGGER.Infof("exit %s (%s)", s, time.Since(start))
	}
}
