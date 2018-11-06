package lager

import (
	"encoding/json"
	"github.com/go-mesh/openlogging"
)

//LogLevel is a user defined variable of type int
type LogLevel int

const (
	//DEBUG is a constant of user defined type LogLevel
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

//FormatLogLevel is a function which returns string format of log level
func FormatLogLevel(x LogLevel) string {
	var level string
	switch x {
	case DEBUG:
		level = "DEBUG"
	case INFO:
		level = "INFO"
	case WARN:
		level = "WARN"
	case ERROR:
		level = "ERROR"
	case FATAL:
		level = "FATAL"
	}
	return level
}

//MarshalJSON is a function which returns data in JSON format
func (x LogLevel) MarshalJSON() ([]byte, error) {
	// var level string
	var level = FormatLogLevel(x)
	return json.Marshal(level)
}

//LogFormat is a struct which stores details about log
type LogFormat struct {
	LogLevel  LogLevel         `json:"level"`
	Timestamp string           `json:"timestamp"`
	File      string           `json:"file"`
	Message   string           `json:"msg"`
	Data      openlogging.Tags `json:"data,omitempty"`
}

//ToJSON which converts data of log file in to JSON file
func (log LogFormat) ToJSON() ([]byte, error) {
	return json.Marshal(log)
}
