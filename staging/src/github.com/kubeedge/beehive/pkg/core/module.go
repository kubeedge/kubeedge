package core

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/context"
)

const (
	tryReadKeyTimes = 5
)

// Module interface
type Module interface {
	Name() string
	Group() string
	Start(c *context.Context)
	Cleanup()
}

var (
	// Modules map
	modules        map[string]Module
	enabledModules map[string]struct{}
)

func init() {
	modules = make(map[string]Module)
	enabledModules = make(map[string]struct{})
}

// Register register module
func Register(m Module) {
	if isModuleEnabled(m.Name()) {
		modules[m.Name()] = m
		klog.Infof("Module %v registered", m.Name())
	} else {
		klog.Warningf("Module %v is disabled, forbid register", m.Name())
	}
}

func SetEnabledModules(names ...string) {
	for _, n := range names {
		enabledModules[n] = struct{}{}
		klog.Infof("Set Module %v enabled", n)
	}
}

func isModuleEnabled(m string) bool {
	_, ok := enabledModules[m]
	return ok
}

// GetModules gets modules map
func GetModules() map[string]Module {
	return modules
}
