package core

import (
	"time"

	klog "k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core/socket"
)

// RestartType is restart policy type
type RestartType string

const (
	// RestartPolicyAlways always restart
	RestartTypeAlways RestartType = "Always"
	// RestartPolicyOnFailure on failure restart
	RestartTypeOnFailure RestartType = "OnFailure"
)

const (
	DefaultRestartIntervalLimit = 30 * time.Second
	DefaultIntervalSecond       = 1
)

// ModuleRestartPolicy is module restart policy
type ModuleRestartPolicy struct {
	// RestartType is the type of restart policy
	RestartType RestartType
	// Retries indicates the number of restarts. If the value is 0, will always restart.
	Retries int32
	// IntervalSecond is the interval seconds between each restart. Default is 1 second.
	IntervalSecond int32
	// IntervalTimeGrowthRate is the growth rate of the time interval between restarts.
	// The value must be greater than 1, otherwise it will be ignored.
	// The interval between each restart is: IntervalTime * IntervalTimeGrowthRate.
	IntervalTimeGrowthRate float64
	// RestartIntervalLimit is the maximum time interval between restarts. Default is 30 seconds.
	RestartIntervalLimit time.Duration
	// ErrorHandler if the Retries is set and reaches the maximum, this method is used to customize error handling.
	// The default handling is to print the error log.
	ErrorHandler func(err error)
}

// Module interface
type Module interface {
	// Name returns the module name.
	Name() string
	// Group returns the module group.
	Group() string
	// Enable returns the module enabled.
	Enable() bool
	// Start starts the module. This is a runtime function, so error handling is left to the user's control.
	// Normally, you can print the error in the Start() and interrupt the module process with the 'return' keyword.
	// You also can define restart handling by RestartPolicy(), and use panic() to trigger 'OnFailure' restart policy.
	Start()
	// RestartPolicy returns the module's restart policy.
	// If the module does not require a restart policy, return nil.
	RestartPolicy() *ModuleRestartPolicy
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

// Register register module
// if not passed in parameter opts, default contextType is "channel"
func Register(m Module, opts ...string) {
	info := &ModuleInfo{
		module:      m,
		contextType: common.MsgCtxTypeChannel,
		remote:      false,
	}

	if len(opts) > 0 && opts[0] == common.MsgCtxTypeUS {
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

// ModuleInfo represent a module info
type ModuleInfo struct {
	contextType string
	remote      bool
	module      Module
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
