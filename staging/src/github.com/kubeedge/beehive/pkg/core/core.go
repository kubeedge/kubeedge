package core

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	klog "k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

// StartModules starts modules that are registered
func StartModules() {
	// only register channel mode, if we want to use socket mode, we should also pass in common.MsgCtxTypeUS parameter
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	modules := GetModules()

	for name, module := range modules {
		m := common.ModuleInfo{
			ModuleName: name,
			ModuleType: module.contextType,
			// the below field ModuleSocket is only required for using socket.
			ModuleSocket: common.ModuleSocket{
				IsRemote: module.remote,
			},
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
func localModuleKeeper(m *ModuleInfo) {
	ctx := beehiveContext.GetContext()
	policy := m.module.RestartPolicy()

	// policy is nil, just start module
	if policy == nil {
		m.module.Start()
		return
	}

	var (
		restartCount int32
		intervalTime time.Duration
	)

	if policy.IntervalSecond > 0 {
		intervalTime = time.Duration(policy.IntervalSecond) * time.Second
	} else {
		intervalTime = time.Duration(DefaultIntervalSecond) * time.Second
	}

	for {
		err := startModule(m)
		if err == nil && policy.RestartType == RestartTypeOnFailure {
			return
		}
		if err != nil {
			klog.Errorf("module %s start failed, err: %v", m.module.Name(), err)
		}
		restartCount++

		if policy.Retries > 0 && restartCount > policy.Retries {
			klog.Infof("module %s restart limit has been reached, count: %d, policy.Retries: %d",
				m.module.Name(), restartCount-1, policy.Retries)
			if policy.ErrorHandler != nil {
				policy.ErrorHandler(err)
			}
			return
		}

		select {
		case <-ctx.Done():
			klog.Infof("module %s shutdown", m.module.Name())
			return
		case <-time.After(intervalTime):
		}
		intervalTime = calculateIntervalTime(intervalTime,
			policy.RestartIntervalLimit, policy.IntervalTimeGrowthRate)
	}
}

func startModule(m *ModuleInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	m.module.Start()
	return
}

func calculateIntervalTime(curr time.Duration, limit time.Duration, growthRate float64) (intervalTime time.Duration) {
	intervalTime = curr
	if growthRate <= 1 {
		return
	}
	if limit == 0 {
		limit = DefaultRestartIntervalLimit
	}
	if curr == limit {
		return
	}
	intervalTime = time.Duration(float64(intervalTime) * growthRate)
	if intervalTime > limit {
		intervalTime = limit
	}
	return
}
