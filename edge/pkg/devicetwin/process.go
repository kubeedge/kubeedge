package devicetwin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
		klog.Errorf("Not found action, msg key info is: source: %s, resource: %s, operation: %s", message.Msg.GetSource(), message.Msg.Router.Resource, message.Msg.Router.Operation)
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
		klog.Infof("Not found deal module for msg, msg action is %s", message.Action)
		return errors.New("Not found deal module for msg")
	}

	return nil
}

func initEventActionMap() {
	EventActionMap = make(map[string]map[string]string)
	EventActionMap[dtcommon.MemETPrefix] = make(map[string]string)
	EventActionMap[dtcommon.DeviceETPrefix] = make(map[string]string)
	EventActionMap[dtcommon.MemETPrefix][dtcommon.MemETGetSuffix] = dtcommon.MemGet
	EventActionMap[dtcommon.MemETPrefix][dtcommon.MemETAddSuffix] = dtcommon.MemAdded
	EventActionMap[dtcommon.MemETPrefix][dtcommon.MemETDeleteSuffix] = dtcommon.MemDeleted

	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.TwinETUpdateSuffix] = dtcommon.TwinUpdate // device side will send this topic
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.TwinETCloudSyncSuffix] = dtcommon.TwinCloudSync
	EventActionMap[dtcommon.DeviceETPrefix][dtcommon.TwinETGetSuffix] = dtcommon.TwinGet
}

func initActionModuleMap() {
	ActionModuleMap = make(map[string]string)
	//membership twin device event , not lifecycle event
	ActionModuleMap[dtcommon.MemGet] = dtcommon.MemModule
	ActionModuleMap[dtcommon.MemAdded] = dtcommon.MemModule
	ActionModuleMap[dtcommon.MemDeleted] = dtcommon.MemModule

	ActionModuleMap[dtcommon.TwinGet] = dtcommon.TwinModule
	ActionModuleMap[dtcommon.TwinUpdate] = dtcommon.TwinModule
	ActionModuleMap[dtcommon.TwinCloudSync] = dtcommon.TwinModule

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
		deviceKey := &dtclient.DevicePrimaryKey{
			Name:      device.Name,
			Namespace: device.Namespace,
		}
		err := SyncDeviceFromSqlite(context, deviceKey)
		if err != nil {
			continue
		}
	}
	return nil
}

//SyncDeviceFromSqlite sync device from sqlite
func SyncDeviceFromSqlite(context *dtcontext.DTContext, deviceKey *dtclient.DevicePrimaryKey) error {
	deviceID := deviceKey.Namespace + "/" + deviceKey.Name
	klog.Infof("Sync device detail info from DB of device %s", deviceID)
	_, exist := context.GetDevice(deviceID)
	if !exist {
		var deviceMutex sync.Mutex
		context.DeviceMutex.Store(deviceID, &deviceMutex)
	}

	defer context.Unlock(deviceID)
	context.Lock(deviceID)

	devices, err := dtclient.QueryDeviceByKey(*deviceKey)
	if err != nil {
		klog.Errorf("query device failed: %v", err)
		return err
	}
	if len(*devices) <= 0 {
		return errors.New("Not found device from db")
	}

	deviceTwinPrimaryKey := dtclient.DeviceTwinPrimaryKey{
		DeviceName:      deviceKey.Name,
		DeviceNamespace: deviceKey.Namespace,
	}

	deviceTwin, err := dtclient.QueryDeviceTwin(&deviceTwinPrimaryKey)
	if err != nil {
		klog.Errorf("query device twin failed: %v", err)
		return err
	}

	cacheDevice := dtclient.GetK8sDeviceFromDeviceTwin(*deviceTwin)
	context.DeviceList.Store(deviceID, cacheDevice)

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
			// device use namespace+name as device ID
			if splitString[0] == "$hw" && splitString[1] == "events" && splitString[2] == "device" {
				identity = splitString[idLoc] + "/" + splitString[idLoc+1]
			} else {
				identity = splitString[idLoc]
			}

			loc := strings.Index(topic, identity)
			nextLoc := loc + len(identity)
			prefix := topic[0:loc]
			suffix := topic[nextLoc:]

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
		fmt.Printf("receive message AAAAAAAAAAAAAAA, resource is %v", message.Msg.Router.Resource)
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
		if strings.Contains(message.Msg.Router.Resource, "membership/added") {
			// add device
			message.Action = dtcommon.MemAdded
			return true
		} else if strings.Contains(message.Msg.Router.Resource, "membership/deleted") {
			// delete device
			message.Action = dtcommon.MemDeleted
			return true
		} else if strings.Contains(message.Msg.Router.Resource, "twin/cloud_updated") {
			message.Action = dtcommon.TwinCloudSync
			resources := strings.Split(message.Msg.Router.Resource, "/")
			if len(resources) > 2 {
				message.Identity = resources[1] + "/" + resources[2]
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
	moduleNames := []string{dtcommon.MemModule, dtcommon.TwinModule, dtcommon.CommModule}
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
