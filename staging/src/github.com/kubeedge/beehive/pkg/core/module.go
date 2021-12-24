package core

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core/socket"
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
	modules         map[string]*ModuleInfo
	disabledModules map[string]*ModuleInfo
)

func init() {
	modules = make(map[string]*ModuleInfo)
	disabledModules = make(map[string]*ModuleInfo)
}

// ModuleInfo represent a module info
type ModuleInfo struct {
	contextType string
	remote      bool
	module      Module
}

// Register register module
// if not passed in parameter opts, default contextType is "channel"
func Register(m Module, opts ...string) {
	info := &ModuleInfo{
		module:      m,
		contextType: common.MsgCtxTypeChannel,
		remote:      false,
	}

	if len(opts) > 0 {
		info.contextType = opts[0]
		info.remote = true
	}

	if m.Enable() {
		modules[m.Name()] = info
		klog.Infof("Module %s registered successfully", m.Name())
	} else {
		disabledModules[m.Name()] = info
		klog.Warningf("Module %v is disabled, do not register", m.Name())
	}
}

// GetModules gets modules map
func GetModules() map[string]*ModuleInfo {
	return modules
}

// GetModule gets module
func (m *ModuleInfo) GetModule() Module {
	return m.module
}

// GetModuleExchange return module exchange
func GetModuleExchange() *socket.ModuleExchange {
	exchange := socket.ModuleExchange{
		Groups: make(map[string][]string),
	}
	for name, moduleInfo := range modules {
		exchange.Modules = append(exchange.Modules, name)
		group := moduleInfo.module.Group()
		exchange.Groups[group] = append(exchange.Groups[group], name)
	}
	return &exchange
}
