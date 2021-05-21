package dtmanager

import (
	"encoding/json"
	"errors"
	"strings"
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
	//deviceActionCallBack map for action to callback
	deviceActionCallBack map[string]CallBack
)

//DeviceWorker deal device event
type DeviceWorker struct {
	Worker
	Group string
}

//Start worker
func (dw DeviceWorker) Start() {
	initDeviceActionCallBack()
	for {
		select {
		case msg, ok := <-dw.ReceiverChan:
			if !ok {
				return
			}
			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := deviceActionCallBack[dtMsg.Action]; exist {
					_, err := fn(dw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("DeviceModule deal %s event failed: %v", dtMsg.Action, err)
					}
				} else {
					klog.Errorf("DeviceModule deal %s event failed, not found callback", dtMsg.Action)
				}
			}
		case v, ok := <-dw.HeartBeatChan:
			if !ok {
				return
			}
			if err := dw.DTContexts.HeartBeat(dw.Group, v); err != nil {
				return
			}
		}
	}
}

func initDeviceActionCallBack() {
	deviceActionCallBack = make(map[string]CallBack)
	deviceActionCallBack[dtcommon.DeviceUpdated] = dealDeviceAttrUpdate
	deviceActionCallBack[dtcommon.DeviceStateUpdate] = dealDeviceStateUpdate
}

func dealDeviceStateUpdate(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	updatedDevice, err := dttype.UnmarshalDeviceUpdate(message.Content.([]byte))
	if err != nil {
		klog.Errorf("Unmarshal device info failed, err: %#v", err)
		return nil, err
	}
	deviceID := resource
	defer context.Unlock(deviceID)
	context.Lock(deviceID)
	doc, docExist := context.DeviceList.Load(deviceID)
	if !docExist {
		return nil, nil
	}
	device, ok := doc.(*dttype.Device)
	if !ok {
		return nil, nil
	}

	// state refers to definition in mappers-go/pkg/common/const.go
	state := strings.ToLower(updatedDevice.State)
	switch state {
	case "online", "offline", "ok", "unknown", "disconnected":
	default:
		return nil, nil
	}
	lastOnline := time.Now().Format("2006-01-02 15:04:05")
	for i := 1; i <= dtcommon.RetryTimes; i++ {
		err = dtclient.UpdateDeviceField(device.ID, "state", updatedDevice.State)
		err = dtclient.UpdateDeviceField(device.ID, "last_online", lastOnline)
		if err == nil {
			break
		}
		time.Sleep(dtcommon.RetryInterval)
	}
	if err != nil {

	}
	device.State = updatedDevice.State
	device.LastOnline = lastOnline
	payload, err := dttype.BuildDeviceState(dttype.BuildBaseMessage(), *device)
	if err != nil {

	}
	topic := dtcommon.DeviceETPrefix + device.ID + dtcommon.DeviceETStateUpdateSuffix + "/result"
	context.Send(device.ID,
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, payload))

	msgResource := "device/" + device.ID + dtcommon.DeviceETStateUpdateSuffix
	context.Send(deviceID,
		dtcommon.SendToCloud,
		dtcommon.CommModule,
		context.BuildModelMessage("resource", "", msgResource, model.UpdateOperation, string(payload)))
	return nil, nil
}

func dealDeviceAttrUpdate(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	updatedDevice, err := dttype.UnmarshalDeviceUpdate(message.Content.([]byte))
	if err != nil {
		klog.Errorf("Unmarshal device info failed, err: %#v", err)
		return nil, err
	}

	deviceID := resource

	context.Lock(deviceID)
	UpdateDeviceAttr(context, deviceID, updatedDevice.Attributes, dttype.BaseMessage{EventID: updatedDevice.EventID}, 0)
	context.Unlock(deviceID)
	return nil, nil
}

//UpdateDeviceAttr update device attributes
func UpdateDeviceAttr(context *dtcontext.DTContext, deviceID string, attributes map[string]*dttype.MsgAttr, baseMessage dttype.BaseMessage, dealType int) (interface{}, error) {
	klog.Infof("Begin to update attributes of the device %s", deviceID)
	var err error
	doc, docExist := context.DeviceList.Load(deviceID)
	if !docExist {
		return nil, nil
	}
	Device, ok := doc.(*dttype.Device)
	if !ok {
		return nil, nil
	}
	dealAttrResult := DealMsgAttr(context, Device.ID, attributes, dealType)
	add, delete, update, result := dealAttrResult.Add, dealAttrResult.Delete, dealAttrResult.Update, dealAttrResult.Result
	if len(add) != 0 || len(delete) != 0 || len(update) != 0 {
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			err = dtclient.DeviceAttrTrans(add, delete, update)
			if err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		now := time.Now().UnixNano() / 1e6
		baseMessage.Timestamp = now

		if err != nil {
			SyncDeviceFromSqlite(context, deviceID)
			klog.Errorf("Update device failed due to writing sql error: %v", err)
		} else {
			klog.Infof("Send update attributes of device %s event to edge app", deviceID)
			payload, err := dttype.BuildDeviceAttrUpdate(baseMessage, result)
			if err != nil {
				//todo
				klog.Errorf("Build device attribute update failed: %v", err)
			}
			topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.DeviceETUpdatedSuffix
			context.Send(deviceID, dtcommon.SendToEdge, dtcommon.CommModule,
				context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, payload))
		}
	}

	return nil, nil
}

//DealMsgAttr get diff,0:update, 1:detail
func DealMsgAttr(context *dtcontext.DTContext, deviceID string, msgAttrs map[string]*dttype.MsgAttr, dealType int) dttype.DealAttrResult {
	device, ok := context.GetDevice(deviceID)
	if !ok {

	}
	attrs := device.Attributes
	if attrs == nil {
		device.Attributes = make(map[string]*dttype.MsgAttr)
		attrs = device.Attributes
	}
	add := make([]dtclient.DeviceAttr, 0)
	deletes := make([]dtclient.DeviceDelete, 0)
	update := make([]dtclient.DeviceAttrUpdate, 0)
	result := make(map[string]*dttype.MsgAttr)

	for key, msgAttr := range msgAttrs {
		if attr, exist := attrs[key]; exist {
			if msgAttr == nil && dealType == 0 {
				if *attr.Optional {
					deletes = append(deletes, dtclient.DeviceDelete{DeviceID: deviceID, Name: key})
					result[key] = nil
					delete(attrs, key)
				}
				continue
			}
			isChange := false
			cols := make(map[string]interface{})
			result[key] = &dttype.MsgAttr{}
			if strings.Compare(attr.Value, msgAttr.Value) != 0 {
				attr.Value = msgAttr.Value

				cols["value"] = msgAttr.Value
				result[key].Value = msgAttr.Value

				isChange = true
			}
			if msgAttr.Metadata != nil {
				msgMetaJSON, _ := json.Marshal(msgAttr.Metadata)
				attrMetaJSON, _ := json.Marshal(attr.Metadata)
				if strings.Compare(string(msgMetaJSON), string(attrMetaJSON)) != 0 {
					cols["attr_type"] = msgAttr.Metadata.Type
					meta := dttype.CopyMsgAttr(msgAttr)
					attr.Metadata = meta.Metadata
					msgAttr.Metadata.Type = ""
					metaJSON, _ := json.Marshal(msgAttr.Metadata)
					cols["metadata"] = string(metaJSON)
					msgAttr.Metadata.Type = cols["attr_type"].(string)
					result[key].Metadata = meta.Metadata
					isChange = true
				}
			}
			if msgAttr.Optional != nil {
				if *msgAttr.Optional != *attr.Optional && *attr.Optional {
					optional := *msgAttr.Optional
					cols["optional"] = optional
					attr.Optional = &optional
					result[key].Optional = &optional
					isChange = true
				}
			}
			if isChange {
				update = append(update, dtclient.DeviceAttrUpdate{DeviceID: deviceID, Name: key, Cols: cols})
			} else {
				delete(result, key)
			}
		} else {
			deviceAttr := dttype.MsgAttrToDeviceAttr(key, msgAttr)
			deviceAttr.DeviceID = deviceID
			deviceAttr.Value = msgAttr.Value
			if msgAttr.Optional != nil {
				optional := *msgAttr.Optional
				deviceAttr.Optional = optional
			}
			if msgAttr.Metadata != nil {
				//todo
				deviceAttr.AttrType = msgAttr.Metadata.Type
				msgAttr.Metadata.Type = ""
				metaJSON, _ := json.Marshal(msgAttr.Metadata)
				msgAttr.Metadata.Type = deviceAttr.AttrType
				deviceAttr.Metadata = string(metaJSON)
			}
			add = append(add, deviceAttr)
			attrs[key] = msgAttr
			result[key] = msgAttr
		}
	}
	if dealType > 0 {
		for key := range attrs {
			if _, exist := msgAttrs[key]; !exist {
				deletes = append(deletes, dtclient.DeviceDelete{DeviceID: deviceID, Name: key})
				result[key] = nil
			}
		}
		for _, v := range deletes {
			delete(attrs, v.Name)
		}
	}
	return dttype.DealAttrResult{Add: add, Delete: deletes, Update: update, Result: result, Err: nil}
}
