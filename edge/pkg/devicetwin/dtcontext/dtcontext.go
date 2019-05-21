package dtcontext

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var (
	//IsDetail deal detail lock
	IsDetail = false
)

//DTContext context for devicetwin
type DTContext struct {
	GroupID        string
	NodeID         string
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
func InitDTContext(context *context.Context) (*DTContext, error) {
	groupID := ""
	nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
	if err != nil {
		log.LOGGER.Warnf("failed to get node id  for web socket client")
	}
	commChan := make(map[string]chan interface{})
	confirmChan := make(chan interface{}, 1000)
	var modulesHealth sync.Map
	var confirm sync.Map
	var deviceList sync.Map
	var deviceMutex sync.Map
	var mutex sync.RWMutex
	// var deviceVersionList sync.Map

	return &DTContext{
		GroupID:        groupID,
		NodeID:         nodeID,
		CommChan:       commChan,
		ConfirmChan:    confirmChan,
		ConfirmMap:     &confirm,
		ModulesHealth:  &modulesHealth,
		ModulesContext: context,
		DeviceList:     &deviceList,
		DeviceMutex:    &deviceMutex,
		Mutex:          &mutex,
		State:          dtcommon.Disconnected,
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
		log.LOGGER.Infof("%s is healthy %v", dtmName, time.Now().Unix())

	} else if strings.Compare(content.(string), "stop") == 0 {
		log.LOGGER.Infof("%s stop", dtmName)
		return errors.New("stop")
	}
	return nil
}

//GetMutex get mutex
func (dtc *DTContext) GetMutex(deviceID string) (*sync.Mutex, bool) {
	v, mutexExist := dtc.DeviceMutex.Load(deviceID)
	if !mutexExist {
		log.LOGGER.Errorf("GetMutex device %s not exist", deviceID)
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
	if ok {
		return true
	}
	return false
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
