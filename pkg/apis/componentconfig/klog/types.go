package klog

//当前日志文件配置
type LogConfig struct {
	//日志文件夹路径
	LogDir string `json:"logDir,omitempty"`
	//日志文件路径(优先级高于LogPath)
	LogFile string `json:"logFile,omitempty"`
	//日志文件最大大小，超过之后自动产生新的日志文件
	LogFileMaxSize int `json:"logFileMaxSize,omitempty"`
	//日志文件数量大小，超过之后自动删除最新产生的日志文件
	LogFileCount    int  `json:"logFileCount,omitempty"`
	LogToStderr     bool `json:"logToStderr,omitempty"`
	AlsoLogToStderr bool `json:"alsoLogToStderr,omitempty"`
}
