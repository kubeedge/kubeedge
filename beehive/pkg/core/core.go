package core

import (
	"os"
	"os/signal"
	"syscall"

	"kubeedge/beehive/pkg/common/log"
	"kubeedge/beehive/pkg/core/context"
)

// start modules that registered
func StartModules() {
	coreContext := context.GetContext(context.MsgCtxTypeChannel)

	modules := GetModules()
	for name, module := range modules {
		//Init the module
		coreContext.AddModule(name)
		//Assemble typeChannels for send2Group
		coreContext.AddModuleGroup(name, module.Group())
		go module.Start(coreContext)
		log.LOGGER.Info("starting module " + name)
	}
}

// if get the special signals, cleanup modules
func GracefulShutdown() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP, syscall.SIGABRT)
	select {
	case s := <-c:
		log.LOGGER.Info("got os signal " + s.String())
		//Cleanup each modules
		modules := GetModules()
		for name, module := range modules {
			log.LOGGER.Info("Cleanup module " + name)
			module.Cleanup()
		}
	}
}

func Run() {
	//Address the module registration and start the core
	StartModules()
	// monitor system singal and shutdown gracefully
	GracefulShutdown()
}
