package core

import (
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

func moduleKeeper(info moduleInfo) {
	for {
		info.module.Start()
		// local modules are always online
		if !info.remote {
			return
		}
		//moduleInfo.module.Cleanup()
		// try to add module for remote modules
		module := common.ModuleInfo{
			ModuleName: info.module.Name(),
			ModuleType: info.moduleType,
			ModuleSocket: common.ModuleSocket{
				IsRemote: info.remote,
			},
		}
		beehiveContext.AddModule(module)
		beehiveContext.AddModuleGroup(info.module.Name(), info.module.Group())
	}
}

// StartModules starts modules that are registered
func StartModules() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	modules := GetModules()
	for name, moduleInfo := range modules {
		var m common.ModuleInfo
		if moduleInfo.moduleType == "" || moduleInfo.moduleType == common.MsgCtxTypeChannel {
			m = common.ModuleInfo{
				ModuleName: name,
				ModuleType: moduleInfo.moduleType,
			}
		} else {
			m = common.ModuleInfo{
				ModuleName: name,
				ModuleType: moduleInfo.moduleType,
				// the below field ModuleSocket is only required for using socket.
				ModuleSocket: common.ModuleSocket{
					IsRemote: moduleInfo.remote,
				},
			}
		}

		beehiveContext.AddModule(m)
		beehiveContext.AddModuleGroup(name, moduleInfo.module.Group())

		go moduleKeeper(moduleInfo)
		klog.Infof("starting module %s", name)
	}
}

// GracefulShutdown is if it gets the special signals it does modules cleanup
func GracefulShutdown() {
	c := make(chan os.Signal)
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
