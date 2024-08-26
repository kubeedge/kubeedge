package core

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

// StartModules starts modules that are registered
func StartModules() {
	// only register channel mode, if we want to use socket mode, we should also pass in common.MsgCtxTypeUS parameter
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	modules := GetModules()

	for name, module := range modules {
		var m common.ModuleInfo
		switch module.contextType {
		case common.MsgCtxTypeChannel:
			m = common.ModuleInfo{
				ModuleName: name,
				ModuleType: module.contextType,
			}
		case common.MsgCtxTypeUS:
			m = common.ModuleInfo{
				ModuleName: name,
				ModuleType: module.contextType,
				// the below field ModuleSocket is only required for using socket.
				ModuleSocket: common.ModuleSocket{
					IsRemote: module.remote,
				},
			}
		default:
			klog.Exitf("unsupported context type: %s", module.contextType)
		}

		beehiveContext.AddModule(&m)
		beehiveContext.AddModuleGroup(name, module.module.Group())

		if module.remote {
			go moduleKeeper(name, module, m)
		} else {
			go localModuleKeeper(module)
		}

		klog.Infof("starting module %s", name)
	}
}

// GracefulShutdown is if it gets the special signals it does modules cleanup
func GracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP, syscall.SIGABRT)
	s := <-c
	klog.Infof("Get os signal %v", s.String())

	// Cleanup each modules
	beehiveContext.Cancel()
	modules := GetModules()
	for name := range modules {
		klog.Infof("Cleanup module %v", name)
		beehiveContext.Cleanup(name)
	}
}

// Run starts the modules and in the end does module cleanup
func Run() {
	// Address the module registration and start the core
	StartModules()
	// monitor system signal and shutdown gracefully
	GracefulShutdown()
}

func moduleKeeper(name string, moduleInfo *ModuleInfo, m common.ModuleInfo) {
	for {
		moduleInfo.module.Start()
		// local modules are always online
		if !moduleInfo.remote {
			return
		}
		// try to add module for remote modules
		beehiveContext.AddModule(&m)
		beehiveContext.AddModuleGroup(name, moduleInfo.module.Group())
	}
}

// localModuleKeeper starts and tries to keep module running when module exited.
// Call EnableModuleRestart() to enable auto-restarting feature in alpha version.
func localModuleKeeper(m *ModuleInfo) {
	if !moduleRestartEnabled {
		m.module.Start()
		return
	}

	ctx := beehiveContext.GetContext()
	backoffDuration := time.Second

	// do if module exits
	afterFunc := func() {
		if r := recover(); r != nil {
			klog.Errorf("module %s panicking: %v", m.module.Name(), r)
		}
		klog.Errorf("module %s exited, will restart in %ds", m.module.Name(), int(backoffDuration.Seconds()))
	}

	for {
		func() {
			defer afterFunc()
			m.module.Start()
		}()

		select {
		case <-ctx.Done():
			klog.Infof("module %s shutdown", m.module.Name())
			return
		case <-time.After(backoffDuration):
		}

		if backoffDuration < 30*time.Second {
			backoffDuration *= 2
			if backoffDuration > 30*time.Second {
				backoffDuration = 30 * time.Second
			}
		}
	}
}
