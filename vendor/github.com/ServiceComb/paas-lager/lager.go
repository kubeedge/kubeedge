package log

import (
	"fmt"
	"github.com/go-chassis/paas-lager/third_party/forked/cloudfoundry/lager"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	//DEBUG is a constant of string type
	DEBUG = "DEBUG"
	//INFO is constant for info level log
	INFO = "INFO"
	//WARN is constant for warn level log
	WARN = "WARN"
	//ERROR is constant for error level log
	ERROR = "ERROR"
	//FATAL is constant for fail level log
	FATAL = "FATAL"
)

//Config is a struct which stores details for maintaining logs
type Config struct {
	LoggerLevel    string   `yaml:"loggerLevel"`
	LoggerFile     string   `yaml:"loggerFile"`
	Writers        []string `yaml:"writers,flow"`
	LogFormatText  bool     `yaml:"logFormatText"`
	EnableRsyslog  bool
	RsyslogNetwork string
	RsyslogAddr    string
}

var config = DefaultConfig()
var m sync.RWMutex

//Writers is a map
var Writers = make(map[string]io.Writer)

//RegisterWriter is used to register a io writer
func RegisterWriter(name string, writer io.Writer) {
	m.Lock()
	Writers[name] = writer
	m.Unlock()
}

//DefaultConfig is a function which retuns config object with default configuration
func DefaultConfig() *Config {
	return &Config{
		LoggerLevel:    INFO,
		LoggerFile:     "",
		EnableRsyslog:  false,
		RsyslogNetwork: "udp",
		RsyslogAddr:    "127.0.0.1:5140",
		LogFormatText:  false,
	}
}

//Init is a function which initializes all config struct variables
func Init(c Config) {
	if c.LoggerLevel != "" {
		config.LoggerLevel = c.LoggerLevel
	}

	if c.LoggerFile != "" {
		config.LoggerFile = c.LoggerFile
		config.Writers = append(config.Writers, "file")
	}

	if c.EnableRsyslog {
		config.EnableRsyslog = c.EnableRsyslog
	}

	if c.RsyslogNetwork != "" {
		config.RsyslogNetwork = c.RsyslogNetwork
	}

	if c.RsyslogAddr != "" {
		config.RsyslogAddr = c.RsyslogAddr
	}
	if len(c.Writers) == 0 {
		config.Writers = append(config.Writers, "stdout")

	} else {
		config.Writers = c.Writers
	}
	config.LogFormatText = c.LogFormatText
	RegisterWriter("stdout", os.Stdout)
	var file io.Writer
	var err error
	if config.LoggerFile != "" {
		file, err = os.OpenFile(config.LoggerFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			panic(err)
		}

	}
	for _, sink := range config.Writers {
		if sink == "file" {
			if file == nil {
				log.Panic("Must set file path")
			}
			RegisterWriter("file", file)
		}
	}
}

//NewLogger is a function
func NewLogger(component string) lager.Logger {
	return NewLoggerExt(component, component)
}

//NewLoggerExt is a function which is used to write new logs
func NewLoggerExt(component string, appGUID string) lager.Logger {
	var lagerLogLevel lager.LogLevel
	switch strings.ToUpper(config.LoggerLevel) {
	case DEBUG:
		lagerLogLevel = lager.DEBUG
	case INFO:
		lagerLogLevel = lager.INFO
	case WARN:
		lagerLogLevel = lager.WARN
	case ERROR:
		lagerLogLevel = lager.ERROR
	case FATAL:
		lagerLogLevel = lager.FATAL
	default:
		panic(fmt.Errorf("unknown logger level: %s", config.LoggerLevel))
	}
	logger := lager.NewLoggerExt(component, config.LogFormatText)
	for _, sink := range config.Writers {

		writer, ok := Writers[sink]
		if !ok {
			log.Panic("Unknow writer: ", sink)
		}
		sink := lager.NewReconfigurableSink(lager.NewWriterSink(sink, writer, lager.DEBUG), lagerLogLevel)
		logger.RegisterSink(sink)
	}

	return logger
}
