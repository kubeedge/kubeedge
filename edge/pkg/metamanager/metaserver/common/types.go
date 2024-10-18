package common

type RestartInfo struct {
	Namespace string
	PodNames  []string
}

type LogsInfo struct {
	Namespace                    string `query:"-"`
	PodName                      string `query:"-"`
	ContainerName                string `query:"-"`
	TailLines                    string `query:"tailLines"`
	Follow                       string `query:"follow"`
	InsecureSkipTLSVerifyBackend string `query:"insecureSkipTLSVerifyBackend"`
	LimitBytes                   string `query:"limitBytes"`
	Pretty                       string `query:"pretty"`
	SinceSeconds                 string `query:"sinceSeconds"`
	Timestamps                   string `query:"timestamps"`
}

type ExecInfo struct {
	Namespace string
	PodName   string
	Container string
	Commands  []string
	Stdin     bool
	Stdout    bool
	Stderr    bool
	TTY       bool
}
