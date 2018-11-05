# Logging Utility for Go-Chassis

A structured logger for Go

### Usage
Create logger
```
log.Init(paas_lager.Config{
        LoggerLevel:   loggerLevel,
        LoggerFile:    loggerFile,
        LogFormatText:  false,
})

logger := paas_lager.NewLogger(component)
```

* LoggerLevel: 日志级别由低到高分别为 DEBUG, INFO, WARN, ERROR, FATAL 共5个级别，这里设置的级别是日志输出的最低级别，只有不低于该级别的日志才会输出
* LoggerFile: 输出日志的文件名，为空则输出到 os.Stdout
* LogFormatText: 设定日志的输出格式是 json 还是 plaintext

Create logger with multiple sinker
```go
	log.Init(log.Config{
		LoggerLevel:   "DEBUG",
		LoggerFile:    "test.log",
		EnableRsyslog: false,
		LogFormatText: false,
		Writers:       []string{"file", "stdout"},
	})

	logger := log.NewLogger("example")
```
Run log rotate
```go
rotate.RunLogRotate("test.log", &rotate.RotateConfig{}, logger)
```

Custom your own sinker
```go
type w struct {
}

func (w *w) Write(p []byte) (n int, err error) {
	fmt.Print("fake")
	return 0, nil
}
func main() {
	log.RegisterWriter("test", &w{})
}

```
See [Example](examples/main.go)