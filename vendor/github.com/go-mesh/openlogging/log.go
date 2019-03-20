package openlogging

import (
	"log"
)

const (
	info  = "INFO: "
	debug = "DEBUG: "
	error = "ERROR: "
	warn  = "WARN: "
	fatal = "FATAL: "
)

type golog struct {
}

func (l golog) Debug(message string, opts ...Option) {
	log.Println(debug + message)
}

func (l golog) Info(message string, opts ...Option) {
	log.Println(info + message)
}
func (l golog) Warn(message string, opts ...Option) {
	log.Println(warn + message)
}
func (l golog) Error(message string, opts ...Option) {
	log.Println(error + message)
}
func (l golog) Fatal(message string, opts ...Option) {
	log.Panic(fatal + message)
}

func (l golog) Debugf(template string, args ...interface{}) {
	log.Printf(debug+template, args)
}
func (l golog) Infof(template string, args ...interface{}) {
	log.Printf(info+template, args)
}
func (l golog) Warnf(template string, args ...interface{}) {
	log.Printf(warn+template, args)
}
func (l golog) Errorf(template string, args ...interface{}) {
	log.Printf(error+template, args)
}
func (l golog) Fatalf(template string, args ...interface{}) {
	log.Panicf(fatal+template, args)
}
