package lager

import "sync/atomic"
//ReconfigurableSink is a struct
type ReconfigurableSink struct {
	sink Sink

	minLogLevel int32
}
//NewReconfigurableSink is a function which returns struct object
func NewReconfigurableSink(sink Sink, initialMinLogLevel LogLevel) *ReconfigurableSink {
	return &ReconfigurableSink{
		sink: sink,

		minLogLevel: int32(initialMinLogLevel),
	}
}
//Log is a method which returns log level and log
func (sink *ReconfigurableSink) Log(level LogLevel, log []byte) {
	minLogLevel := LogLevel(atomic.LoadInt32(&sink.minLogLevel))

	if level < minLogLevel {
		return
	}

	sink.sink.Log(level, log)
}
//SetMinLevel is a function which sets minimum log level
func (sink *ReconfigurableSink) SetMinLevel(level LogLevel) {
	atomic.StoreInt32(&sink.minLogLevel, int32(level))
}
//GetMinLevel is a method which gets minimum log level
func (sink *ReconfigurableSink) GetMinLevel() LogLevel {
	return LogLevel(atomic.LoadInt32(&sink.minLogLevel))
}
