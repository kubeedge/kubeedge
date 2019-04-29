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

	"github.com/kubeedge/beehive/pkg/common/config"
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

	LOGGER = log.NewLogger("github.com/kubeedge/kubeedge")
	writers := logConfig.Writers
	for _, value := range writers {
		if value == "file" {
			rotate.RunLogRotate(logConfig.LoggerFile, &rotate.RotateConfig{}, LOGGER)
			break
		}
	}

	LOGGER.Debug("init logger...")
}

// Trace will output the execution time for a given function.
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

// Debugf Default Debugf
func Debugf(format string, args ...interface{}) {
	LOGGER.Debugf(format, args...)
}

// Debug Default Debug
func Debug(action string) {
	LOGGER.Debug(action)
}

// Infof Default Infof
func Infof(format string, args ...interface{}) {
	LOGGER.Infof(format, args...)
}

// Info Default Info
func Info(action string) {
	LOGGER.Info(action)
}

// Warnf Default Warnf
func Warnf(format string, args ...interface{}) {
	LOGGER.Warnf(format, args...)
}

// Warn Default Warn
func Warn(action string) {
	LOGGER.Warn(action)
}

// Errorf Default Errorf
func Errorf(format string, args ...interface{}) {
	LOGGER.Errorf(format, args...)
}

// Error Default Error
func Error(action string) {
	LOGGER.Error(action)
}

// Fatalf Default Fatalf
func Fatalf(format string, args ...interface{}) {
	LOGGER.Fatalf(format, args...)
}

// Fatal Default Fatal
func Fatal(action string) {
	LOGGER.Fatal(action)
}
