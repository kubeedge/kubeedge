package core

import (
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/context"
)

// StartModules starts modules that are registered
func StartModules() {
	coreContext := context.GetContext(context.MsgCtxTypeChannel)

	modules := GetModules()
	for name, module := range modules {
		//Init the module
		coreContext.AddModule(name)
		//Assemble typeChannels for send2Group
		coreContext.AddModuleGroup(name, module.Group())
		go module.Start(coreContext)
		klog.Infof("Starting module %v", name)
	}
}

// GracefulShutdown is if it gets the special signals it does modules cleanup
func GracefulShutdown() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP, syscall.SIGABRT)
	select {
	case s := <-c:
		klog.Infof("Get os signal %v", s.String())
		//Cleanup each modules
		modules := GetModules()
		for name, module := range modules {
			klog.Infof("Cleanup module %v", name)
			module.Cleanup()
		}
	}
}

//Run starts the modules and in the end does module cleanup
func Run() {
	//Address the module registration and start the core
	StartModules()
	// monitor system signal and shutdown gracefully
	GracefulShutdown()
}
