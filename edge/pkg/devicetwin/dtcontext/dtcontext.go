package dtcontext

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	deviceconfig "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/config"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

//DTContext context for devicetwin
type DTContext struct {
	GroupID        string
	NodeName       string
	CommChan       map[string]chan interface{}
	ConfirmChan    chan interface{}
	ConfirmMap     *sync.Map
	ModulesHealth  *sync.Map
	ModulesContext *context.Context
	DeviceList     *sync.Map
	DeviceMutex    *sync.Map
	Mutex          *sync.RWMutex
	// DBConn *dtclient.Conn
	State string
}

//InitDTContext init dtcontext
func InitDTContext() (*DTContext, error) {
	return &DTContext{
		GroupID:       "",
		NodeName:      deviceconfig.Config.NodeName,
		CommChan:      make(map[string]chan interface{}),
		ConfirmChan:   make(chan interface{}, 1000),
		ConfirmMap:    &sync.Map{},
		ModulesHealth: &sync.Map{},
		DeviceList:    &sync.Map{},
		DeviceMutex:   &sync.Map{},
		Mutex:         &sync.RWMutex{},
		State:         dtcommon.Disconnected,
	}, nil
}

//CommTo communicate
func (dtc *DTContext) CommTo(dtmName string, content interface{}) error {
	if v, exist := dtc.CommChan[dtmName]; exist {
		v <- content
		return nil
	}
	return errors.New("Not found chan to communicate")
}

//HeartBeat hearbeat to dtcontroller
func (dtc *DTContext) HeartBeat(dtmName string, content interface{}) error {
	if strings.Compare(content.(string), "ping") == 0 {
		dtc.ModulesHealth.Store(dtmName, time.Now().Unix())
		klog.V(3).Infof("%s is healthy %v", dtmName, time.Now().Unix())
	} else if strings.Compare(content.(string), "stop") == 0 {
		klog.Infof("%s stop", dtmName)
		return errors.New("stop")
	}
	return nil
}

//GetMutex get mutex
func (dtc *DTContext) GetMutex(deviceID string) (*sync.Mutex, bool) {
	v, mutexExist := dtc.DeviceMutex.Load(deviceID)
	if !mutexExist {
		klog.Errorf("GetMutex device %s not exist", deviceID)
		return nil, false
	}
	mutex, isMutex := v.(*sync.Mutex)
	if isMutex {
		return mutex, true
	}
	return nil, false
}

//Lock get the lock of the device
func (dtc *DTContext) Lock(deviceID string) bool {
	deviceMutex, ok := dtc.GetMutex(deviceID)
	if ok {
		dtc.Mutex.RLock()
		deviceMutex.Lock()
		return true
	}
	return false
}

//Unlock remove the lock of the device
func (dtc *DTContext) Unlock(deviceID string) bool {
	deviceMutex, ok := dtc.GetMutex(deviceID)
	if ok {
		deviceMutex.Unlock()
		dtc.Mutex.RUnlock()
		return true
	}
	return false
}

// LockAll get all lock
func (dtc *DTContext) LockAll() {
	dtc.Mutex.Lock()
}

// UnlockAll get all lock
func (dtc *DTContext) UnlockAll() {
	dtc.Mutex.Unlock()
}

//IsDeviceExist judge device is exist
func (dtc *DTContext) IsDeviceExist(deviceID string) bool {
	_, ok := dtc.DeviceList.Load(deviceID)
	return ok
}

//GetDevice get device
func (dtc *DTContext) GetDevice(deviceID string) (*dttype.Device, bool) {
	d, ok := dtc.DeviceList.Load(deviceID)
	if ok {
		if device, isDevice := d.(*dttype.Device); isDevice {
			return device, true
		}
		return nil, false
	}
	return nil, false
}

//Send send result
func (dtc *DTContext) Send(identity string, action string, module string, msg *model.Message) error {
	dtMsg := &dttype.DTMessage{
		Action:   action,
		Identity: identity,
		Type:     module,
		Msg:      msg}
	return dtc.CommTo(module, dtMsg)
}

//BuildModelMessage build mode messages
func (dtc *DTContext) BuildModelMessage(group string, parentID string, resource string, operation string, content interface{}) *model.Message {
	msg := model.NewMessage(parentID)
	msg.BuildRouter(modules.TwinGroup, group, resource, operation)
	msg.Content = content
	return msg
}
