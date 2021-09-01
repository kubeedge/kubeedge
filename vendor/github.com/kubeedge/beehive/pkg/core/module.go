package core

import (
	"k8s.io/klog/v2"
)

const (
	tryReadKeyTimes = 5
)

// Module interface
type Module interface {
	Name() string
	Group() string
	Start()
	Enable() bool
}

var (
	// Modules map
	modules         map[string]Module
	disabledModules map[string]Module
)

func init() {
	modules = make(map[string]Module)
	disabledModules = make(map[string]Module)
}

// Register register module
func Register(m Module) {
	if m.Enable() {
		modules[m.Name()] = m
		klog.Infof("Module %v registered successfully", m.Name())
	} else {
		disabledModules[m.Name()] = m
		klog.Warningf("Module %v is disabled, do not register", m.Name())
	}
}

// GetModules gets modules map
func GetModules() map[string]Module {
	return modules
}
