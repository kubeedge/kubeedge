package tracing

import (
	"fmt"

	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-mesh/openlogging"
	"github.com/opentracing/opentracing-go"
)

// TracerFuncMap saves NewTracer func
// key: impl name
// val: tracer new func
var TracerFuncMap = make(map[string]NewTracer)

// NewTracer is the func to return global tracer
type NewTracer func(o map[string]string) (opentracing.Tracer, error)

//InstallTracer install new opentracing tracer
func InstallTracer(name string, f NewTracer) {
	TracerFuncMap[name] = f
	openlogging.Info("installed tracing plugin: " + name)

}

// GetTracerFunc get NewTracer
func GetTracerFunc(name string) (NewTracer, error) {
	tracer, ok := TracerFuncMap[name]
	if !ok {
		return nil, fmt.Errorf("not supported tracer [%s]", name)
	}
	return tracer, nil
}

// Init initialize the global tracer
func Init() error {
	openlogging.Info("Tracing enabled. Start to init tracer.")
	if config.GlobalDefinition.Tracing.Tracer == "" {
		config.GlobalDefinition.Tracing.Tracer = "zipkin"
	}
	f, err := GetTracerFunc(config.GlobalDefinition.Tracing.Tracer)
	if err != nil {
		openlogging.Warn("can not load any opentracing plugin, lost distributed tracing function")
		return nil
	}
	tracer, err := f(config.GlobalDefinition.Tracing.Settings)
	if err != nil {
		return err
	}
	opentracing.SetGlobalTracer(tracer)
	return nil
}
