package fileutil

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	//ChassisConfDir is constant of type string
	ChassisConfDir = "CHASSIS_CONF_DIR"
	//ChassisHome is constant of type string
	ChassisHome = "CHASSIS_HOME"
	//SchemaDirectory is constant of type string
	SchemaDirectory = "schema"
)

const (
	//Global is a constant of type string
	Global = "chassis.yaml"
	//LoadBalancing is constant of type string
	LoadBalancing = "load_balancing.yaml"
	//RateLimiting is constant of type string
	RateLimiting = "rate_limiting.yaml"
	//Definition is constant of type string
	Definition = "microservice.yaml"
	//Hystric is constant of type string
	Hystric = "circuit_breaker.yaml"
	//PaasLager is constant of type string
	PaasLager = "lager.yaml"
	//TLS is constant of type string
	TLS = "tls.yaml"
	//Monitoring is constant of type string
	Monitoring = "monitoring.yaml"
	//Auth is constant of type string
	Auth = "auth.yaml"
	//Tracing is constant of type string
	Tracing = "tracing.yaml"
	//Router is constant of type string
	Router = "router.yaml"
)

var configDir string
var homeDir string
var once sync.Once

//GetWorkDir is a function used to get the working directory
func GetWorkDir() (string, error) {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return wd, nil
}

func initDir() {
	if h := os.Getenv(ChassisHome); h != "" {
		homeDir = h
	} else {
		wd, err := GetWorkDir()
		if err != nil {
			panic(err)
		}
		homeDir = wd
	}

	// set conf dir, CHASSIS_CONF_DIR has highest priority
	if confDir := os.Getenv(ChassisConfDir); confDir != "" {
		configDir = confDir
	} else {
		// CHASSIS_HOME has second most high priority
		configDir = filepath.Join(homeDir, "conf")
	}
}

//ChassisHomeDir is function used to get the home directory of chassis
func ChassisHomeDir() string {
	once.Do(initDir)
	return homeDir
}

//GetConfDir is a function used to get the configuration directory
func GetConfDir() string {
	initDir()
	return configDir
}

//CircuitBreakerConfigPath is a function used to join .yaml file name with configuration path
func CircuitBreakerConfigPath() string {
	return filepath.Join(GetConfDir(), Hystric)
}

//GetDefinition is a function used to join .yaml file name with configuration path
func GetDefinition() string {
	return filepath.Join(GetConfDir(), Definition)
}

//LoadBalancingConfigPath is a function used to join .yaml file name with configuration directory
func LoadBalancingConfigPath() string {
	return filepath.Join(GetConfDir(), LoadBalancing)
}

//RateLimitingFile is a function used to join .yaml file name with configuration directory
func RateLimitingFile() string {
	return filepath.Join(GetConfDir(), RateLimiting)
}

//TLSConfigPath is a function used to join .yaml file name with configuration directory
func TLSConfigPath() string {
	return filepath.Join(GetConfDir(), TLS)
}

//MonitoringConfigPath is a function used to join .yaml file name with configuration directory
func MonitoringConfigPath() string {
	return filepath.Join(GetConfDir(), Monitoring)
}

//MicroserviceDefinition is a function used to join .yaml file name with configuration directory
func MicroserviceDefinition(microserviceName string) string {
	return filepath.Join(GetConfDir(), microserviceName, Definition)
}

//MicroServiceConfigPath is a function used to join .yaml file name with configuration directory
func MicroServiceConfigPath() string {
	return filepath.Join(GetConfDir(), Definition)
}

//GlobalConfigPath is a function used to join .yaml file name with configuration directory
func GlobalConfigPath() string {
	return filepath.Join(GetConfDir(), Global)
}

//LogConfigPath is a function used to join .yaml file name with configuration directory
func LogConfigPath() string {
	return filepath.Join(GetConfDir(), PaasLager)
}

//RouterConfigPath is a function used to join .yaml file name with configuration directory
func RouterConfigPath() string {
	return filepath.Join(GetConfDir(), Router)
}

//AuthConfigPath is a function used to join .yaml file name with configuration directory
func AuthConfigPath() string {
	return filepath.Join(GetConfDir(), Auth)
}

//TracingPath is a function used to join .yaml file name with configuration directory
func TracingPath() string {
	return filepath.Join(GetConfDir(), Tracing)
}

//SchemaDir is a function used to join .yaml file name with configuration path
func SchemaDir(microserviceName string) string {
	return filepath.Join(GetConfDir(), microserviceName, SchemaDirectory)
}
