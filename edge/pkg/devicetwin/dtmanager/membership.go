package dtmanager

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var (
	//memActionCallBack map for action to callback
	memActionCallBack map[string]CallBack
)

//MemWorker deal membership event
type MemWorker struct {
	Worker
	Group string
}

//Start worker
func (mw MemWorker) Start() {
	initMemActionCallBack()
	for {
		select {
		case msg, ok := <-mw.ReceiverChan:
			if !ok {
				return
			}
			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := memActionCallBack[dtMsg.Action]; exist {
					_, err := fn(mw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("MemModule deal %s event failed: %v", dtMsg.Action, err)
					}
				} else {
					klog.Errorf("MemModule deal %s event failed, not found callback", dtMsg.Action)
				}
			}

		case v, ok := <-mw.HeartBeatChan:
			if !ok {
				return
			}
			if err := mw.DTContexts.HeartBeat(mw.Group, v); err != nil {
				return
			}
		}
	}
}

func initMemActionCallBack() {
	memActionCallBack = make(map[string]CallBack)
	memActionCallBack[dtcommon.MemGet] = dealMembershipGet
	memActionCallBack[dtcommon.MemUpdated] = dealMembershipUpdate
	memActionCallBack[dtcommon.MemDetailResult] = dealMembershipDetail
}
func getRemoveList(context *dtcontext.DTContext, devices []dttype.Device) []dttype.Device {
	var toRemove []dttype.Device
	context.DeviceList.Range(func(key interface{}, value interface{}) bool {
		isExist := false
		for _, v := range devices {
			if strings.Compare(v.ID, key.(string)) == 0 {
				isExist = true
				break
			}
		}
		if !isExist {
			toRemove = append(toRemove, dttype.Device{ID: key.(string)})
		}
		return true
	})
	return toRemove
}
func dealMembershipDetail(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.Info("Deal node detail info")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	contentData, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("assertion failed")
	}

	devices, err := dttype.UnmarshalMembershipDetail(contentData)
	if err != nil {
		klog.Errorf("Unmarshal membership info failed , err: %#v", err)
		return nil, err
	}

	baseMessage := dttype.BaseMessage{EventID: devices.EventID}
	defer context.UnlockAll()
	context.LockAll()
	var toRemove []dttype.Device
	isDelta := false
	addDevice(context, devices.Devices, baseMessage, isDelta)
	toRemove = getRemoveList(context, devices.Devices)

	if len(toRemove) != 0 {
		removeDevice(context, toRemove, baseMessage, isDelta)
	}
	klog.Info("Deal node detail info successful")
	return nil, nil
}

func dealMembershipUpdate(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.Infof("Membership event")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	contentData, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("assertion failed")
	}

	updateEdgeGroups, err := dttype.UnmarshalMembershipUpdate(contentData)
	if err != nil {
		klog.Errorf("Unmarshal membership info failed , err: %#v", err)
		return nil, err
	}

	baseMessage := dttype.BaseMessage{EventID: updateEdgeGroups.EventID}
	if updateEdgeGroups.AddDevices != nil && len(updateEdgeGroups.AddDevices) > 0 {
		//add device
		addDevice(context, updateEdgeGroups.AddDevices, baseMessage, false)
	}
	if updateEdgeGroups.RemoveDevices != nil && len(updateEdgeGroups.RemoveDevices) > 0 {
		// delete device
		removeDevice(context, updateEdgeGroups.RemoveDevices, baseMessage, false)
	}
	return nil, nil
}

func dealMembershipGet(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.Infof("MEMBERSHIP EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	contentData, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("assertion failed")
	}

	dealMembershipGetInner(context, contentData)
	return nil, nil
}

// addDevice add device to the edge group
func addDevice(context *dtcontext.DTContext, toAdd []dttype.Device, baseMessage dttype.BaseMessage, delta bool) {
	klog.Infof("Add devices to edge group")
	if !delta {
		baseMessage.EventID = ""
	}
	if len(toAdd) == 0 {
		return
	}
	dealType := 0
	if !delta {
		dealType = 1
	}
	for _, device := range toAdd {
		//if device has existed, step out
		deviceInstance, isDeviceExist := context.GetDevice(device.ID)
		if isDeviceExist {
			if delta {
				klog.Errorf("Add device %s failed, has existed", device.ID)
				continue
			}
			UpdateDeviceAttr(context, device.ID, device.Attributes, baseMessage, dealType)
			DealDeviceTwin(context, device.ID, baseMessage.EventID, device.Twin, dealType)
			//todo sync twin
			continue
		}

		var deviceMutex sync.Mutex
		context.DeviceMutex.Store(device.ID, &deviceMutex)

		if delta {
			context.Lock(device.ID)
		}

		deviceInstance = &dttype.Device{ID: device.ID, Name: device.Name, Description: device.Description, State: device.State}
		context.DeviceList.Store(device.ID, deviceInstance)

		//write to sqlite
		var err error
		adds := make([]dtclient.Device, 0)
		addAttr := make([]dtclient.DeviceAttr, 0)
		addTwin := make([]dtclient.DeviceTwin, 0)
		adds = append(adds, dtclient.Device{
			ID:          device.ID,
			Name:        device.Name,
			Description: device.Description,
			State:       device.State})
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			err = dtclient.AddDeviceTrans(adds, addAttr, addTwin)
			if err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}

		if err != nil {
			klog.Errorf("Add device %s failed due to some error ,err: %#v", device.ID, err)
			context.DeviceList.Delete(device.ID)
			context.Unlock(device.ID)
			continue
			//todo
		}
		if device.Twin != nil {
			klog.Infof("Add device twin during first adding device %s", device.ID)
			DealDeviceTwin(context, device.ID, baseMessage.EventID, device.Twin, dealType)
		}

		if device.Attributes != nil {
			klog.Infof("Add device attr during first adding device %s", device.ID)
			UpdateDeviceAttr(context, device.ID, device.Attributes, baseMessage, dealType)
		}
		topic := dtcommon.MemETPrefix + context.NodeName + dtcommon.MemETUpdateSuffix
		baseMessage := dttype.BuildBaseMessage()
		addDeviceDevices := make([]dttype.Device, 0)
		addDeviceDevices = append(addDeviceDevices, device)
		addDeviceResult := dttype.MembershipUpdate{BaseMessage: baseMessage, AddDevices: addDeviceDevices}
		result, err := dttype.MarshalMembershipUpdate(addDeviceResult)
		if err != nil {

		} else {
			context.Send("",
				dtcommon.SendToEdge,
				dtcommon.CommModule,
				context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
		}
		if delta {
			context.Unlock(device.ID)
		}
	}
}

// removeDevice remove device from the edge group
func removeDevice(context *dtcontext.DTContext, toRemove []dttype.Device, baseMessage dttype.BaseMessage, delta bool) {
	klog.Infof("Begin to remove devices")
	if !delta {
		baseMessage.EventID = ""
	}
	for _, device := range toRemove {
		//update sqlite
		_, deviceExist := context.GetDevice(device.ID)
		if !deviceExist {
			klog.Errorf("Remove device %s failed, not existed", device.ID)
			continue
		}
		if delta {
			context.Lock(device.ID)
		}
		deletes := make([]string, 0)
		deletes = append(deletes, device.ID)
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			err := dtclient.DeleteDeviceTrans(deletes)
			if err != nil {
				klog.Errorf("Delete document of device %s failed at %d time, err: %#v", device.ID, i, err)
			} else {
				klog.Infof("Delete document of device %s successful", device.ID)
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		//todo
		context.DeviceList.Delete(device.ID)
		context.DeviceMutex.Delete(device.ID)
		if delta {
			context.Unlock(device.ID)
		}
		topic := dtcommon.MemETPrefix + context.NodeName + dtcommon.MemETUpdateSuffix
		baseMessage := dttype.BuildBaseMessage()
		RemoveDevices := make([]dttype.Device, 0)
		RemoveDevices = append(RemoveDevices, device)
		deleteResult := dttype.MembershipUpdate{BaseMessage: baseMessage, RemoveDevices: RemoveDevices}
		result, err := dttype.MarshalMembershipUpdate(deleteResult)
		if err != nil {

		} else {
			context.Send("",
				dtcommon.SendToEdge,
				dtcommon.CommModule,
				context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
		}

		klog.Infof("Remove device %s successful", device.ID)
	}
}

// dealMembershipGetInner deal get membership event
func dealMembershipGetInner(context *dtcontext.DTContext, payload []byte) error {
	klog.Info("Deal getting membership event")
	result := []byte("")
	edgeGet, err := dttype.UnmarshalBaseMessage(payload)
	para := dttype.Parameter{}
	now := time.Now().UnixNano() / 1e6
	if err != nil {
		klog.Errorf("Unmarshal get membership info %s failed , err: %#v", string(payload), err)
		para.Code = dtcommon.BadRequestCode
		para.Reason = fmt.Sprintf("Unmarshal get membership info %s failed , err: %#v", string(payload), err)
		var jsonErr error
		result, jsonErr = dttype.BuildErrorResult(para)
		if jsonErr != nil {
			klog.Errorf("Unmarshal error result error, err: %v", jsonErr)
		}
	} else {
		para.EventID = edgeGet.EventID
		var devices []*dttype.Device
		context.DeviceList.Range(func(key interface{}, value interface{}) bool {
			device, ok := value.(*dttype.Device)
			if !ok {

			} else {
				devices = append(devices, device)
			}
			return true
		})

		payload, err := dttype.BuildMembershipGetResult(dttype.BaseMessage{EventID: edgeGet.EventID, Timestamp: now}, devices)
		if err != nil {
			klog.Errorf("Marshal membership failed while deal get membership ,err: %#v", err)
		} else {
			result = payload
		}
	}
	topic := dtcommon.MemETPrefix + context.NodeName + dtcommon.MemETGetResultSuffix
	klog.Infof("Deal getting membership successful and send the result")

	context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))

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

	devices, err := dtclient.QueryDevice("id", deviceID)
	if err != nil {
		klog.Errorf("query device attr failed: %v", err)
		return err
	}
	if len(*devices) == 0 {
		return errors.New("Not found device")
	}
	dbDoc := (*devices)[0]

	deviceAttr, err := dtclient.QueryDeviceAttr("deviceid", deviceID)
	if err != nil {
		klog.Errorf("query device attr failed: %v", err)
		return err
	}

	deviceTwin, err := dtclient.QueryDeviceTwin("deviceid", deviceID)
	if err != nil {
		klog.Errorf("query device twin failed: %v", err)
		return err
	}

	context.DeviceList.Store(deviceID, &dttype.Device{
		ID:          deviceID,
		Name:        dbDoc.Name,
		Description: dbDoc.Description,
		State:       dbDoc.State,
		LastOnline:  dbDoc.LastOnline,
		Attributes:  dttype.DeviceAttrToMsgAttr(*deviceAttr),
		Twin:        dttype.DeviceTwinToMsgTwin(*deviceTwin)})

	return nil
}
