package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/lager"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/pkg/runtime"
	"github.com/go-mesh/openlogging"
)

// DefaultRetryTime default retry time
const DefaultRetryTime = 10 * time.Second

// HeartbeatTask heart beat task struct
type HeartbeatTask struct {
	ServiceID  string
	InstanceID string
	Time       time.Time
	Running    bool
}

// HeartbeatService heartbeat service
type HeartbeatService struct {
	instances map[string]*HeartbeatTask
	shutdown  bool
	mux       sync.Mutex
}

// Start start the heartbeat system
func (s *HeartbeatService) Start() {
	s.shutdown = false
	defer s.Stop()

	s.run()
}

// Stop stop the heartbeat system
func (s *HeartbeatService) Stop() {
	s.shutdown = true
}

// AddTask add new micro-service instance to the heartbeat system
func (s *HeartbeatService) AddTask(microServiceID, microServiceInstanceID string) {
	key := fmt.Sprintf("%s/%s", microServiceID, microServiceInstanceID)
	lager.Logger.Infof("Add HB task, task:%s", key)
	s.mux.Lock()
	if _, ok := s.instances[key]; !ok {
		s.instances[key] = &HeartbeatTask{
			ServiceID:  microServiceID,
			InstanceID: microServiceInstanceID,
			Time:       time.Now(),
		}
	}
	s.mux.Unlock()
}

// RemoveTask remove micro-service instance from the heartbeat system
func (s *HeartbeatService) RemoveTask(microServiceID, microServiceInstanceID string) {
	key := fmt.Sprintf("%s/%s", microServiceID, microServiceInstanceID)
	s.mux.Lock()
	delete(s.instances, key)
	s.mux.Unlock()
}

// RefreshTask refresh heartbeat for micro-service instance
func (s *HeartbeatService) RefreshTask(microServiceID, microServiceInstanceID string) {
	key := fmt.Sprintf("%s/%s", microServiceID, microServiceInstanceID)
	s.mux.Lock()
	if _, ok := s.instances[key]; ok {
		s.instances[key].Time = time.Now()
	}
	s.mux.Unlock()
}

// toggleTask toggle task
func (s *HeartbeatService) toggleTask(microServiceID, microServiceInstanceID string, running bool) {
	key := fmt.Sprintf("%s/%s", microServiceID, microServiceInstanceID)
	s.mux.Lock()
	if _, ok := s.instances[key]; ok {
		s.instances[key].Running = running
	}
	s.mux.Unlock()
}

// DoHeartBeat do heartbeat for each instance
func (s *HeartbeatService) DoHeartBeat(microServiceID, microServiceInstanceID string) {
	s.toggleTask(microServiceID, microServiceInstanceID, true)
	_, err := DefaultRegistrator.Heartbeat(microServiceID, microServiceInstanceID)
	if err != nil {
		lager.Logger.Errorf("Run Heartbeat fail: %s", err)
		s.RemoveTask(microServiceID, microServiceInstanceID)
		s.RetryRegister(microServiceID, microServiceInstanceID)
	}
	s.RefreshTask(microServiceID, microServiceInstanceID)
	s.toggleTask(microServiceID, microServiceInstanceID, false)
}

// run runs the heartbeat system
func (s *HeartbeatService) run() {
	for !s.shutdown {
		s.mux.Lock()
		endTime := time.Now()
		for _, v := range s.instances {
			if v.Running {
				continue
			}
			if endTime.Sub(v.Time) >= common.DefaultHBInterval*time.Second {
				go s.DoHeartBeat(v.ServiceID, v.InstanceID)
			}
		}
		s.mux.Unlock()
		time.Sleep(time.Second)
	}
}

// RetryRegister retrying to register micro-service, and instance
func (s *HeartbeatService) RetryRegister(sid, iid string) {
	for {
		openlogging.Info("try to re-register")
		_, err := DefaultServiceDiscoveryService.GetAllMicroServices()
		if err != nil {
			lager.Logger.Errorf("DefaultRegistrator is not healthy %s", err)
			continue
		}
		if _, e := DefaultServiceDiscoveryService.GetMicroService(sid); e != nil {
			err = s.ReRegisterSelfMSandMSI()
		} else {
			err = reRegisterSelfMSI(sid, iid)
		}
		if err == nil {
			break
		}
		time.Sleep(DefaultRetryTime)
	}
	openlogging.Warn("Re-register self success")
}

// ReRegisterSelfMSandMSI 重新注册微服务和实例
func (s *HeartbeatService) ReRegisterSelfMSandMSI() error {
	err := RegisterMicroservice()
	if err != nil {
		lager.Logger.Errorf("The reRegisterSelfMSandMSI() startMicroservice failed: %s", err)
		return err
	}

	err = RegisterMicroserviceInstances()
	if err != nil {
		lager.Logger.Errorf("The reRegisterSelfMSandMSI() startInstances failed: %s", err)
		return err
	}
	return nil
}

// reRegisterSelfMSI 只重新注册实例
func reRegisterSelfMSI(sid, iid string) error {
	eps, err := MakeEndpointMap(config.GlobalDefinition.Cse.Protocols)
	if err != nil {
		return err
	}
	if len(InstanceEndpoints) != 0 {
		eps = InstanceEndpoints
	}
	microServiceInstance := &MicroServiceInstance{
		InstanceID:   iid,
		EndpointsMap: eps,
		HostName:     runtime.HostName,
		Status:       common.DefaultStatus,
		Metadata:     runtime.InstanceMD,
	}
	instanceID, err := DefaultRegistrator.RegisterServiceInstance(sid, microServiceInstance)
	if err != nil {
		lager.Logger.Errorf("RegisterInstance failed: %s", err)
		return err
	}

	value, ok := SelfInstancesCache.Get(microServiceInstance.ServiceID)
	if !ok {
		lager.Logger.Warnf("RegisterMicroServiceInstance get SelfInstancesCache failed, microServiceID/instanceID: %s/%s", sid, instanceID)
	}
	instanceIDs, ok := value.([]string)
	if !ok {
		lager.Logger.Warnf("RegisterMicroServiceInstance type asserts failed, microServiceID/instanceID: %s/%s", sid, instanceID)
	}
	var isRepeat bool
	for _, va := range instanceIDs {
		if va == instanceID {
			isRepeat = true
		}
	}
	if !isRepeat {
		instanceIDs = append(instanceIDs, instanceID)
	}
	SelfInstancesCache.Set(microServiceInstance.ServiceID, instanceIDs, 0)
	lager.Logger.Infof("RegisterMicroServiceInstance success, microServiceID/instanceID: %s/%s.", sid, instanceID)

	return nil
}
