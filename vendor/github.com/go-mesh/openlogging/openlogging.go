package openlogging

var logger Logger

func SetLogger(l Logger) {
	logger = l
}
func GetLogger() Logger {
	return logger
}

func Debug(message string, opts ...Option) {
	opts = append(opts, WithDepth(2))
	logger.Debug(message, opts...)
}
func Info(message string, opts ...Option) {
	opts = append(opts, WithDepth(2))
	logger.Info(message, opts...)
}
func Warn(message string, opts ...Option) {
	opts = append(opts, WithDepth(2))
	logger.Warn(message, opts...)
}
func Error(message string, opts ...Option) {
	opts = append(opts, WithDepth(2))
	logger.Error(message, opts...)
}
func Fatal(message string, opts ...Option) {
	opts = append(opts, WithDepth(2))
	logger.Fatal(message, opts...)
}
func init() {
	logger = &golog{}
}
