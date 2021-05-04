package devicetwin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var (
	//EventActionMap map for event to action
	EventActionMap map[string]map[string]string
	//ActionModuleMap map for action to module
	ActionModuleMap map[string]string
)

//RegisterDTModule register dtmodule
func (dt *DeviceTwin) RegisterDTModule(name string) {
	module := dtmodule.DTModule{
		Name: name,
	}

	dt.DTContexts.CommChan[name] = make(chan interface{}, 128)
	dt.HeartBeatToModule[name] = make(chan interface{}, 128)
	module.InitWorker(dt.DTContexts.CommChan[name], dt.DTContexts.ConfirmChan,
		dt.HeartBeatToModule[name], dt.DTContexts)
	dt.DTModules[name] = module
}

//distributeMsg distribute message to diff module
func (dt *DeviceTwin) distributeMsg(m interface{}) error {
	msg, ok := m.(model.Message)
	if !ok {
		klog.Errorf("Distribute message, msg is nil")
		return errors.New("Distribute message, msg is nil")
	}
	message := dttype.DTMessage{Msg: &msg}
	if message.Msg.GetParentID() != "" {
		klog.Infof("Send msg to the %s module in twin", dtcommon.CommModule)
		confirmMsg := dttype.DTMessage{Msg: model.NewMessage(message.Msg.GetParentID()), Action: dtcommon.Confirm}
		if err := dt.DTContexts.CommTo(dtcommon.CommModule, &confirmMsg); err != nil {
			return err
		}
	}
	if !classifyMsg(&message) {
		return errors.New("Not found action")
	}
	if ActionModuleMap == nil {
		initActionModuleMap()
	}

	if moduleName, exist := ActionModuleMap[message.Action]; exist {
		//how to deal write channel error
		klog.Infof("Send msg to the %s module in twin", moduleName)
		if err := dt.DTContexts.CommTo(moduleName, &message); err != nil {
			return err
		}
	} else {
		klog.Info("Not found deal module for msg")
		return errors.New("Not found deal module for msg")
	}

	return nil
}

func initEventActionMap() {
	EventActionMap = make(map[string]map[string]string)
	EventActionMap[dtcommon.MemETPrefix] = make(map[string]string)
	EventActionMap[dtcommon.DeviceETPrefix] = make(map[string]string)
	EventActionMap[dtcommon.MemETPrefix][dtcommon.MemETDetailResultSuffix] = dtcommon.MemDetailResult
	EventActionMap[dtcommon.MemETPrefix][dtcommon.MemETUpdateSuffix] = dtcommon.MemUpdated
	EventActionMap[dtcommon.MemETPrefix][dtcommon.MemETGetSuffix] = dtcommon.MemGet
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.DeviceETStateGetSuffix] = dtcommon.DeviceStateGet
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.DeviceETUpdatedSuffix] = dtcommon.DeviceUpdated
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.DeviceETStateUpdateSuffix] = dtcommon.DeviceStateUpdate
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.TwinETUpdateSuffix] = dtcommon.TwinUpdate
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.TwinETCloudSyncSuffix] = dtcommon.TwinCloudSync
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.TwinETGetSuffix] = dtcommon.TwinGet
}

func initActionModuleMap() {
	ActionModuleMap = make(map[string]string)
	//membership twin device event , not lifecycle event
	ActionModuleMap[dtcommon.MemDetailResult] = dtcommon.MemModule
	ActionModuleMap[dtcommon.MemGet] = dtcommon.MemModule
	ActionModuleMap[dtcommon.MemUpdated] = dtcommon.MemModule
	ActionModuleMap[dtcommon.TwinGet] = dtcommon.TwinModule
	ActionModuleMap[dtcommon.TwinUpdate] = dtcommon.TwinModule
	ActionModuleMap[dtcommon.TwinCloudSync] = dtcommon.TwinModule
	ActionModuleMap[dtcommon.DeviceUpdated] = dtcommon.DeviceModule
	ActionModuleMap[dtcommon.DeviceStateGet] = dtcommon.DeviceModule
	ActionModuleMap[dtcommon.DeviceStateUpdate] = dtcommon.DeviceModule
	ActionModuleMap[dtcommon.Connected] = dtcommon.CommModule
	ActionModuleMap[dtcommon.Disconnected] = dtcommon.CommModule
	ActionModuleMap[dtcommon.LifeCycle] = dtcommon.CommModule
	ActionModuleMap[dtcommon.Confirm] = dtcommon.CommModule
}

// SyncSqlite sync sqlite
func SyncSqlite(context *dtcontext.DTContext) error {
	klog.Info("Begin to sync sqlite ")
	rows, queryErr := dtclient.QueryDeviceAll()
	if queryErr != nil {
		klog.Errorf("Query sqlite failed while syncing sqlite, err: %#v", queryErr)
		return queryErr
	}
	if rows == nil {
		klog.Info("Query sqlite nil while syncing sqlite")
		return nil
	}
	for _, device := range *rows {
		err := SyncDeviceFromSqlite(context, device.ID)
		if err != nil {
			continue
		}
	}
	return nil
}

//SyncDeviceFromSqlite sync device from sqlite
func SyncDeviceFromSqlite(context *dtcontext.DTContext, deviceID string) error {
	klog.Infof("Sync device detail info from DB of device %s", deviceID)
	_, exist := context.GetDevice(deviceID)
	if !exist {
		var deviceMutex sync.Mutex
		context.DeviceMutex.Store(deviceID, &deviceMutex)
	}

	defer context.Unlock(deviceID)
	context.Lock(deviceID)

	devices, err := dtclient.QueryDevice("id", deviceID)
	if err != nil {
		klog.Errorf("query device failed: %v", err)
		return err
	}
	if len(*devices) <= 0 {
		return errors.New("Not found device from db")
	}
	device := (*devices)[0]

	deviceAttr, err := dtclient.QueryDeviceAttr("deviceid", deviceID)
	if err != nil {
		klog.Errorf("query device attr failed: %v", err)
		return err
	}
	attributes := make([]dtclient.DeviceAttr, 0)
	attributes = append(attributes, *deviceAttr...)

	deviceTwin, err := dtclient.QueryDeviceTwin("deviceid", deviceID)
	if err != nil {
		klog.Errorf("query device twin failed: %v", err)
		return err
	}
	twins := make([]dtclient.DeviceTwin, 0)
	twins = append(twins, *deviceTwin...)

	context.DeviceList.Store(deviceID, &dttype.Device{
		ID:          deviceID,
		Name:        device.Name,
		Description: device.Description,
		State:       device.State,
		LastOnline:  device.LastOnline,
		Attributes:  dttype.DeviceAttrToMsgAttr(attributes),
		Twin:        dttype.DeviceTwinToMsgTwin(twins)})

	return nil
}

func classifyMsg(message *dttype.DTMessage) bool {
	if EventActionMap == nil {
		initEventActionMap()
	}
	var identity string
	var action string
	msgSource := message.Msg.GetSource()
	if strings.Compare(msgSource, "bus") == 0 {
		idLoc := 3
		topic := message.Msg.GetResource()
		topicByte, err := base64.URLEncoding.DecodeString(topic)
		if err != nil {
			return false
		}
		topic = string(topicByte)

		klog.Infof("classify the msg with the topic %s", topic)
		splitString := strings.Split(topic, "/")
		if len(splitString) == 4 {
			if strings.HasPrefix(topic, dtcommon.LifeCycleConnectETPrefix) {
				action = dtcommon.LifeCycle
			} else if strings.HasPrefix(topic, dtcommon.LifeCycleDisconnectETPrefix) {
				action = dtcommon.LifeCycle
			} else {
				return false
			}
		} else {
			identity = splitString[idLoc]
			loc := strings.Index(topic, identity)
			nextLoc := loc + len(identity)
			prefix := topic[0:loc]
			suffix := topic[nextLoc:]
			klog.Infof("%s %s", prefix, suffix)
			if v, exist := EventActionMap[prefix][suffix]; exist {
				action = v
			} else {
				return false
			}
		}
		message.Msg.Content = []byte((message.Msg.Content).(string))
		message.Identity = identity
		message.Action = action
		klog.Infof("Classify the msg to action %s", action)
		return true
	} else if (strings.Compare(msgSource, "edgemgr") == 0) || (strings.Compare(msgSource, "devicecontroller") == 0) {
		switch message.Msg.Content.(type) {
		case []byte:
			klog.Info("Message content type is []byte, no need to marshal again")
		default:
			content, err := json.Marshal(message.Msg.Content)
			if err != nil {
				return false
			}
			message.Msg.Content = content
		}
		if strings.Contains(message.Msg.Router.Resource, "membership/detail") {
			message.Action = dtcommon.MemDetailResult
			return true
		} else if strings.Contains(message.Msg.Router.Resource, "membership") {
			message.Action = dtcommon.MemUpdated
			return true
		} else if strings.Contains(message.Msg.Router.Resource, "twin/cloud_updated") {
			message.Action = dtcommon.TwinCloudSync
			resources := strings.Split(message.Msg.Router.Resource, "/")
			message.Identity = resources[1]
			return true
		} else if strings.Contains(message.Msg.Router.Operation, "updated") {
			resources := strings.Split(message.Msg.Router.Resource, "/")
			if len(resources) == 2 && strings.Compare(resources[0], "device") == 0 {
				message.Action = dtcommon.DeviceUpdated
				message.Identity = resources[1]
			}
			return true
		}
		return false
	} else if strings.Compare(msgSource, "edgehub") == 0 {
		if strings.Compare(message.Msg.Router.Resource, "node/connection") == 0 {
			message.Action = dtcommon.LifeCycle
			return true
		}
		return false
	}
	return false
}

func (dt *DeviceTwin) runDeviceTwin() {
	moduleNames := []string{dtcommon.MemModule, dtcommon.TwinModule, dtcommon.DeviceModule, dtcommon.CommModule}
	for _, v := range moduleNames {
		dt.RegisterDTModule(v)
		go dt.DTModules[v].Start()
	}
	go func() {
		for {
			select {
			case <-beehiveContext.Done():
				klog.Warning("Stop DeviceTwin ModulesContext Receive loop")
				return
			default:
			}
			if msg, ok := beehiveContext.Receive("twin"); ok == nil {
				klog.Info("DeviceTwin receive msg")
				err := dt.distributeMsg(msg)
				if err != nil {
					klog.Warningf("distributeMsg failed: %v", err)
				}
			}
		}
	}()

	for {
		select {
		case <-time.After((time.Duration)(60) * time.Second):
			//range to check whether has bug
			for dtmName := range dt.DTModules {
				health, ok := dt.DTContexts.ModulesHealth.Load(dtmName)
				if ok {
					now := time.Now().Unix()
					if now-health.(int64) > 60*2 {
						klog.Infof("%s health %v is old, and begin restart", dtmName, health)
						go dt.DTModules[dtmName].Start()
					}
				}
			}
			for _, v := range dt.HeartBeatToModule {
				v <- "ping"
			}
		case <-beehiveContext.Done():
			for _, v := range dt.HeartBeatToModule {
				v <- "stop"
			}
			klog.Warning("Stop DeviceTwin ModulesHealth load loop")
			return
		}
	}
}
