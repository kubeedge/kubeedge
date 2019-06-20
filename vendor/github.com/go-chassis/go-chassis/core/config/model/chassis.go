package model

//GlobalCfg chassis.yaml 配置项
type GlobalCfg struct {
	AppID      string            `yaml:"APPLICATION_ID"` //Deprecated
	Cse        CseStruct         `yaml:"cse"`
	Panel      ControlPanel      `yaml:"control"`
	Ssl        map[string]string `yaml:"ssl"`
	Tracing    TracingStruct     `yaml:"tracing"`
	DataCenter *DataCenterInfo   `yaml:"region"`
}

// DataCenterInfo gives data center information
type DataCenterInfo struct {
	Name          string `yaml:"name"`
	Region        string `yaml:"region"`
	AvailableZone string `yaml:"availableZone"`
}

//CseStruct 设置注册中心SC的地址，要开哪些传输协议， 调用链信息等
type CseStruct struct {
	Config          Config                      `yaml:"config"`
	Service         ServiceStruct               `yaml:"service"`
	Protocols       map[string]Protocol         `yaml:"protocols"`
	Handler         HandlerStruct               `yaml:"handler"`
	References      map[string]ReferencesStruct `yaml:"references"` //Deprecated
	FlowControl     FlowControl                 `yaml:"flowcontrol"`
	Monitor         MonitorStruct               `yaml:"monitor"`
	Metrics         MetricsStruct               `yaml:"metrics"`
	Credentials     CredentialStruct            `yaml:"credentials"`
	Transport       Transport                   `yaml:"transport"`
	NoRefreshSchema bool                        `yaml:"noRefreshSchema"`
}

//Transport defines failure
//TODO support TLS config
type Transport struct {
	Failure      map[string]string `yaml:"failure"`
	MaxIdlCons   map[string]int    `yaml:"maxIdleCon"`
	MaxBodyBytes map[string]int64  `yaml:"maxBodyBytes"`
}

// MetricsStruct metrics struct
type MetricsStruct struct {
	APIPath                   string `yaml:"apiPath"`
	FlushInterval             string `yaml:"flushInterval"`
	Enable                    bool   `yaml:"enable"`
	EnableGoRuntimeMetrics    bool   `yaml:"enableGoRuntimeMetrics"`
	EnableCircuitMetrics      bool   `yaml:"enableCircuitMetrics"`
	CircuitMetricsConsumerNum int    `yaml:"circuitMetricsConsumerNum"`
}

// MonitorStruct is the struct for monitoring parameters
type MonitorStruct struct {
	Client MonitorClientStruct `yaml:"client"`
}

// MonitorClientStruct monitor client struct
type MonitorClientStruct struct {
	ServerURI  string                  `yaml:"serverUri"`
	Enable     bool                    `yaml:"enable"`
	UserName   string                  `yaml:"userName"`
	DomainName string                  `yaml:"domainName"`
	APIVersion MonitorAPIVersionStruct `yaml:"api"`
}

// MonitorAPIVersionStruct monitor API version struct
type MonitorAPIVersionStruct struct {
	Version string `yaml:"version"`
}

// FlowControl used to define rate limiting
type FlowControl struct {
	Consumer QPS `yaml:"Consumer"`
	Provider QPS `yaml:"Provider"`
}

// QPS is the struct to define QPS
type QPS struct {
	QPS QPSProps `yaml:"qps"`
}

// QPSProps define rate limiting settings
type QPSProps struct {
	Enabled bool              `yaml:"enabled"`
	Global  map[string]int    `yaml:"global"`
	Limit   map[string]string `yaml:"limit"`
}

// Config represent config center configurations
type Config struct {
	Client ConfigClient `yaml:"client"`
}

// ConfigClient client structure
type ConfigClient struct {
	Type              string                 `yaml:"type"`
	ServerURI         string                 `yaml:"serverUri"`
	TenantName        string                 `yaml:"tenantName"`
	RefreshMode       int                    `yaml:"refreshMode"`
	RefreshInterval   int                    `yaml:"refreshInterval"`
	RefreshPort       string                 `yaml:"refreshPort"`
	Autodiscovery     bool                   `yaml:"autodiscovery"`
	APIVersion        ConfigAPIVersionStruct `yaml:"api"`
	ApolloServiceName string                 `yaml:"serviceName"`
	ApolloEnv         string                 `yaml:"env"`
	ApolloNameSpace   string                 `yaml:"namespace"`
	ApolloToken       string                 `yaml:"token"`
	ClusterName       string                 `yaml:"cluster"`
	Enabled           bool                   `yaml:"enabled"`
}

// ConfigAPIVersionStruct is the structure for configuration API version
type ConfigAPIVersionStruct struct {
	Version string `yaml:"version"`
}

// ReferencesStruct references structure
type ReferencesStruct struct {
	Version   string `yaml:"version"`
	Transport string `yaml:"transport"`
}

// Protocol protocol structure
type Protocol struct {
	Listen       string `yaml:"listenAddress"`
	Advertise    string `yaml:"advertiseAddress"`
	WorkerNumber int    `yaml:"workerNumber"`
	Transport    string `yaml:"transport"`
}

// MicroserviceCfg microservice.yaml 配置项
type MicroserviceCfg struct {
	AppID              string           `yaml:"APPLICATION_ID"`
	Provider           string           `yaml:"Provider"`
	ServiceDescription MicServiceStruct `yaml:"service_description"`
}

// MicServiceStruct 设置微服务的私有属性
type MicServiceStruct struct {
	Name               string              `yaml:"name"`
	Hostname           string              `yaml:"hostname"`
	Version            string              `yaml:"version"`
	Environment        string              `yaml:"environment"`
	Level              string              `yaml:"level"`
	Properties         map[string]string   `yaml:"properties"`
	InstanceProperties map[string]string   `yaml:"instance_properties"`
	ServicePaths       []ServicePathStruct `yaml:"paths"`
	ServicesStatus     string              `yaml:"status"`
}

// ServicePathStruct having info about service path and property
type ServicePathStruct struct {
	Path     string            `yaml:"path"`
	Property map[string]string `yaml:"property"`
}

// HandlerStruct 调用链信息
type HandlerStruct struct {
	Chain ChainStruct `yaml:"chain"`
}

// ChainStruct 调用链信息
type ChainStruct struct {
	Consumer map[string]string `yaml:"Consumer"`
	Provider map[string]string `yaml:"Provider"`
}

// CredentialStruct aksk信息
type CredentialStruct struct {
	AccessKey        string `yaml:"accessKey"`
	SecretKey        string `yaml:"secretKey"`
	AkskCustomCipher string `yaml:"akskCustomCipher"`
	Project          string `yaml:"project"`
}

// TracingStruct tracing structure
type TracingStruct struct {
	Tracer   string            `yaml:"tracer"`
	Settings map[string]string `yaml:"settings"`
}
