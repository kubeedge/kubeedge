package openlogging

type Tags map[string]interface{}

type Options struct {
	Tags Tags
}
type Option func(*Options)

func WithTags(tags Tags) Option {
	return func(o *Options) {
		o.Tags = tags
	}
}

// Logger is a interface for log tool
type Logger interface {
	Debug(message string, opts ...Option)
	Info(message string, opts ...Option)
	Warn(message string, opts ...Option)
	Error(message string, opts ...Option)
	Fatal(message string, opts ...Option)

	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
}
