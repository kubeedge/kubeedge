package lager

import (
	"fmt"
	"github.com/go-mesh/openlogging"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//StackTraceBufferSize is a constant which defines stack track buffer size
const StackTraceBufferSize = 1024 * 100

//Logger is a interface
type Logger interface {
	RegisterSink(Sink)
	SetLogLevel(LogLevel)
	Session(task string, data ...openlogging.Option) Logger
	SessionName() string
	Debug(action string, data ...openlogging.Option)
	Info(action string, data ...openlogging.Option)
	Warn(action string, data ...openlogging.Option)
	Error(action string, data ...openlogging.Option)
	Fatal(action string, data ...openlogging.Option)

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	WithData(openlogging.Tags) Logger
}

type logger struct {
	component     string
	task          string
	sinks         []Sink
	sessionID     string
	nextSession   uint64
	data          openlogging.Tags
	logFormatText bool
}

//NewLoggerExt is a function which returns logger struct object
func NewLoggerExt(component string, isFormatText bool) Logger {
	return &logger{
		component:     component,
		task:          component,
		sinks:         []Sink{},
		data:          openlogging.Tags{},
		logFormatText: isFormatText,
	}
}

//NewLogger is a function used to get new logger object
func NewLogger(component string) Logger {
	return NewLoggerExt(component, true)
}

//RegisterSink is a function used to register sink
func (l *logger) RegisterSink(sink Sink) {
	l.sinks = append(l.sinks, sink)
}

//SessionName is used to get the session name
func (l *logger) SessionName() string {
	return l.task
}

//Session is a function which returns logger details for that session
func (l *logger) Session(task string, opts ...openlogging.Option) Logger {
	opt := &openlogging.Options{}
	for _, o := range opts {
		o(opt)
	}
	sid := atomic.AddUint64(&l.nextSession, 1)

	var sessionIDstr string

	if l.sessionID != "" {
		sessionIDstr = fmt.Sprintf("%s.%d", l.sessionID, sid)
	} else {
		sessionIDstr = fmt.Sprintf("%d", sid)
	}

	return &logger{
		component: l.component,
		task:      fmt.Sprintf("%s.%s", l.task, task),
		sinks:     l.sinks,
		sessionID: sessionIDstr,
		data:      l.baseData(opt.Tags),
	}
}

//WithData which adds data to the logger object
func (l *logger) WithData(data openlogging.Tags) Logger {
	return &logger{
		component: l.component,
		task:      l.task,
		sinks:     l.sinks,
		sessionID: l.sessionID,
		data:      l.baseData(data),
	}
}

// SetLogLevel set logger level, current just support file output
func (l *logger) SetLogLevel(level LogLevel) {
	for _, itf := range l.sinks {
		if s, ok := itf.(*writerSink); ok && s.name != "file" {
			continue
		}
		if s, ok := itf.(*ReconfigurableSink); ok {
			s.SetMinLevel(level)
		}
	}
}


// Find the sink need to log
func (l *logger) activeSinks(loglevel LogLevel) []Sink {
	ss := make([]Sink, len(l.sinks))
	idx := 0
	for _, itf := range l.sinks {
		if s, ok := itf.(*writerSink); ok && loglevel < s.minLogLevel {
			continue
		}
		if s, ok := itf.(*ReconfigurableSink); ok && loglevel < LogLevel(s.minLogLevel) {
			continue
		}
		ss[idx] = itf
		idx++
	}
	return ss[:idx]
}

func (l *logger) log(loglevel LogLevel, action string, opts ...openlogging.Option) {
	ss := l.activeSinks(loglevel)
	if len(ss) == 0 {
		return
	}
	l.logs(ss, loglevel, action, opts...)
}

func (l *logger) logs(ss []Sink, loglevel LogLevel, action string, opts ...openlogging.Option) {
	opt := &openlogging.Options{}
	for _, o := range opts {
		o(opt)
	}
	logData := l.baseData(opt.Tags)

	if loglevel == FATAL {
		stackTrace := make([]byte, StackTraceBufferSize)
		stackSize := runtime.Stack(stackTrace, false)
		stackTrace = stackTrace[:stackSize]

		logData["trace"] = string(stackTrace)
	}

	log := LogFormat{
		Timestamp: currentTimestamp(),
		Message:   action,
		LogLevel:  loglevel,
		Data:      logData,
	}

	// add file, lineno
	addExtLogInfo(&log, opt.Depth)
	var logInfo string
	for _, sink := range l.sinks {
		if l.logFormatText {
			levelstr := FormatLogLevel(log.LogLevel)
			extraData, ok := log.Data["error"].(string)
			if ok && extraData != "" {
				extraData = " error: " + extraData
			}
			logInfo = log.Timestamp + " " + levelstr + " " + log.File + " " + log.Message + extraData
			sink.Log(loglevel, []byte(logInfo))

		} else {
			logInfo, jserr := log.ToJSON()
			if jserr != nil {
				fmt.Printf("[lager] ToJSON() ERROR! action: %s, jserr: %s, log: %+v", action, jserr, log)
				// also output json marshal error event to sink
				log.Data = openlogging.Tags{"Data": fmt.Sprint(logData)}
				jsonerrdata, _ := log.ToJSON()
				sink.Log(ERROR, jsonerrdata)
				continue
			}
			sink.Log(loglevel, logInfo)
		}
	}

	if loglevel == FATAL {
		panic(logInfo)
	}
}

func (l *logger) Debug(action string, data ...openlogging.Option) {
	l.log(DEBUG, action, data...)
}

func (l *logger) Info(action string, data ...openlogging.Option) {
	l.log(INFO, action, data...)
}

func (l *logger) Warn(action string, data ...openlogging.Option) {
	l.log(WARN, action, data...)
}

func (l *logger) Error(action string, data ...openlogging.Option) {
	l.log(ERROR, action, data...)
}

func (l *logger) Fatal(action string, data ...openlogging.Option) {
	l.log(FATAL, action, data...)
}

func (l *logger) logf(loglevel LogLevel, format string, args ...interface{}) {
	ss := l.activeSinks(loglevel)
	if len(ss) == 0 {
		return
	}
	logmsg := fmt.Sprintf(format, args...)
	l.logs(ss, loglevel, logmsg)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	l.logf(DEBUG, format, args...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.logf(INFO, format, args...)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.logf(WARN, format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.logf(ERROR, format, args...)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	l.logf(FATAL, format, args...)
}

func (l *logger) baseData(givenData openlogging.Tags) openlogging.Tags {
	data := openlogging.Tags{}

	for k, v := range l.data {
		data[k] = v
	}

	if len(givenData) > 0 {
		for key, val := range givenData {
			data[key] = val
		}
	}

	if l.sessionID != "" {
		data["session"] = l.sessionID
	}

	return data
}

func currentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000 -07:00")
}

func addExtLogInfo(logf *LogFormat, depth int) {

	for i := 3; i <= 5; i++ {
		_, file, line, ok := runtime.Caller(depth + i)

		if strings.Index(file, "logger.go") > 0 {
			continue
		}

		if ok {
			idx := strings.LastIndex(file, "src")
			switch {
			case idx >= 0:
				logf.File = file[idx+4:]
			default:
				logf.File = file
			}
			// depth: 2
			indexFunc := func(file string) string {
				backup := "/" + file
				lastSlashIndex := strings.LastIndex(backup, "/")
				if lastSlashIndex < 0 {
					return backup
				}
				secondLastSlashIndex := strings.LastIndex(backup[:lastSlashIndex], "/")
				if secondLastSlashIndex < 0 {
					return backup[lastSlashIndex+1:]
				}
				return backup[secondLastSlashIndex+1:]
			}
			logf.File = indexFunc(logf.File) + ":" + strconv.Itoa(line)
		}
		break
	}
}
