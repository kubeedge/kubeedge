package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-mesh/openlogging"
	"github.com/kubeedge/beehive/pkg/common/util"

	log "github.com/go-chassis/paas-lager"
	"github.com/go-chassis/paas-lager/rotate"
	"github.com/go-chassis/paas-lager/third_party/forked/cloudfoundry/lager"
	yaml "gopkg.in/yaml.v2"
)

const (
	LOGCONFIGPATH = "/conf/logging.yaml"
)

// LOGGER var
var LOGGER lager.Logger
var once = sync.Once{}

func InitializeLogger() {
	once.Do(func() {
		logConfig := &log.Config{
			LoggerLevel:   "DEBUG",
			EnableRsyslog: false,
			LogFormatText: true,
			Writers:       []string{"stdout"},
		}

		filename := strings.TrimSuffix(GetConfigDir(), "/") + LOGCONFIGPATH

		//fmt.Println("logging.yaml's path is " + filename)
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
		} else {
			if os.IsNotExist(err) {
				fmt.Printf("No config file exists, using default configurations for logger\n")
			} else {
				fmt.Printf("Got error when reading yaml config file:%v\n", err)
			}
		}

		log.Init(*logConfig)
		logBeehive := log.NewLogger("beehive")
		LOGGER = logBeehive
		openlogging.SetLogger(logBeehive)
		// Set logger rotate
		writers := logConfig.Writers
		for _, value := range writers {
			if value == "file" {
				rotate.RunLogRotate(logConfig.LoggerFile, &rotate.RotateConfig{}, LOGGER)
				break
			}
		}
		LOGGER.Debug("init logger...")
	})
}

func init() {
	InitializeLogger()
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

func GetConfigDir() string {
	configDir := os.Getenv("GOARCHAIUS_CONFIG_PATH")
	if configDir != "" {
		return configDir
	}
	return util.GetCurrentDirectory()
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
