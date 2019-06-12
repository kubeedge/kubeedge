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

//HystrixDefinition is a function used to join .yaml file name with configuration path
func HystrixDefinition() string {
	return filepath.Join(GetConfDir(), Hystric)
}

//GetDefinition is a function used to join .yaml file name with configuration path
func GetDefinition() string {
	return filepath.Join(GetConfDir(), Definition)
}

//GetLoadBalancing is a function used to join .yaml file name with configuration directory
func GetLoadBalancing() string {
	return filepath.Join(GetConfDir(), LoadBalancing)
}

//GetRateLimiting is a function used to join .yaml file name with configuration directory
func GetRateLimiting() string {
	return filepath.Join(GetConfDir(), RateLimiting)
}

//GetTLS is a function used to join .yaml file name with configuration directory
func GetTLS() string {
	return filepath.Join(GetConfDir(), TLS)
}

//GetMonitoring is a function used to join .yaml file name with configuration directory
func GetMonitoring() string {
	return filepath.Join(GetConfDir(), Monitoring)
}

//MicroserviceDefinition is a function used to join .yaml file name with configuration directory
func MicroserviceDefinition(microserviceName string) string {
	return filepath.Join(GetConfDir(), microserviceName, Definition)
}

//GetMicroserviceDesc is a function used to join .yaml file name with configuration directory
func GetMicroserviceDesc() string {
	return filepath.Join(GetConfDir(), Definition)
}

//GlobalDefinition is a function used to join .yaml file name with configuration directory
func GlobalDefinition() string {
	return filepath.Join(GetConfDir(), Global)
}

//PaasLagerDefinition is a function used to join .yaml file name with configuration directory
func PaasLagerDefinition() string {
	return filepath.Join(GetConfDir(), PaasLager)
}

//RouterDefinition is a function used to join .yaml file name with configuration directory
func RouterDefinition() string {
	return filepath.Join(GetConfDir(), Router)
}

//GetAuth is a function used to join .yaml file name with configuration directory
func GetAuth() string {
	return filepath.Join(GetConfDir(), Auth)
}

//GetTracing is a function used to join .yaml file name with configuration directory
func GetTracing() string {
	return filepath.Join(GetConfDir(), Tracing)
}

//SchemaDir is a function used to join .yaml file name with configuration path
func SchemaDir(microserviceName string) string {
	return filepath.Join(GetConfDir(), microserviceName, SchemaDirectory)
}
