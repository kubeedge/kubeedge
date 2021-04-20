package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

const (
	//RestDealType update from mqtt
	RestDealType = 0
	//SyncDealType update form cloud sync
	SyncDealType = 1
	//DetailDealType detail update from cloud
	DetailDealType = 2
	//SyncTwinDeleteDealType twin delete when sync
	SyncTwinDeleteDealType = 3
	//DealActual deal actual
	DealActual = 1
	//DealExpected deal expected
	DealExpected = 0

	stringType = "string"
)

var (
	//twinActionCallBack map for action to callback
	twinActionCallBack         map[string]CallBack
	initTwinActionCallBackOnce sync.Once
)

//TwinWorker deal twin event
type TwinWorker struct {
	Worker
	Group string
}

//Start worker
func (tw TwinWorker) Start() {
	initTwinActionCallBack()
	for {
		select {
		case msg, ok := <-tw.ReceiverChan:
			if !ok {
				return
			}
			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := twinActionCallBack[dtMsg.Action]; exist {
					klog.Infof("important important important action is %s", dtMsg.Action)
					_, err := fn(tw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("TwinModule deal %s event failed", dtMsg.Action)
					}
				} else {
					klog.Errorf("TwinModule deal %s event failed, not found callback", dtMsg.Action)
				}
			}

		case v, ok := <-tw.HeartBeatChan:
			if !ok {
				return
			}
			if err := tw.DTContexts.HeartBeat(tw.Group, v); err != nil {
				return
			}
		}
	}
}

func initTwinActionCallBack() {
	initTwinActionCallBackOnce.Do(func() {
		twinActionCallBack = make(map[string]CallBack)
		twinActionCallBack[dtcommon.TwinUpdate] = dealTwinUpdate
		twinActionCallBack[dtcommon.TwinGet] = dealTwinGet
		twinActionCallBack[dtcommon.TwinCloudSync] = dealTwinSync
	})
}

func dealTwinSync(context *dtcontext.DTContext, deviceID string, msg interface{}) (interface{}, error) {
	klog.Infof("Twin Sync EVENT, device id is %v", deviceID)
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}
	result := []byte("")
	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invalid message content")
	}

	device, err := dttype.UnmarshalDeviceTwinUpdate(content)
	if err != nil {
		klog.Errorf("Unmarshal update request body failed, err: %#v", err)
		dealUpdateResult(context, deviceID, errors.New("Unmarshal update request body failed, Please check the request"), result)
		return nil, err
	}

	context.Lock(deviceID)
	DealDeviceTwin(context, deviceID, device.Status.Twins, SyncDealType)
	context.Unlock(deviceID)
	//todo send ack
	return nil, nil
}

func dealTwinGet(context *dtcontext.DTContext, deviceID string, msg interface{}) (interface{}, error) {
	klog.Infof("Twin Get EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invalid message content")
	}

	DealGetTwin(context, deviceID, content)
	return nil, nil
}

func dealTwinUpdate(context *dtcontext.DTContext, deviceID string, msg interface{}) (interface{}, error) {
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invalid message content")
	}

	if _, isExist := context.GetDevice(deviceID); !isExist {
		dealUpdateResult(context, deviceID, errors.New("Update rejected due to the device is not existed"), []byte(""))
		klog.Errorf("Update twin rejected due to device %v is not exist", deviceID)
		return nil, errors.New("device not exist")
	}

	context.Lock(deviceID)
	Updated(context, deviceID, content)
	context.Unlock(deviceID)
	return nil, nil
}

// Updated update the snapshot
func Updated(context *dtcontext.DTContext, deviceID string, payload []byte) {
	result := []byte("")
	updatedDevice, err := dttype.UnmarshalDeviceTwinUpdate(payload)
	if err != nil {
		klog.Errorf("Unmarshal update request body failed, err: %#v", err)
		dealUpdateResult(context, deviceID, err, result)
		return
	}

	DealDeviceTwin(context, deviceID, updatedDevice.Status.Twins, RestDealType)
}

//DealDeviceTwin deal device twin
func DealDeviceTwin(context *dtcontext.DTContext, deviceID string, twins []v1alpha2.Twin, dealType int) error {
	klog.Infof("Begin to deal device twin of the device %s", deviceID)

	result := []byte("")
	device, isExist := context.GetDevice(deviceID)
	if !isExist {
		klog.Errorf("Update twin rejected due to the device %s is not existed", deviceID)
		dealUpdateResult(context, deviceID, errors.New("Update rejected due to the device is not existed"), result)
		return errors.New("Update rejected due to the device is not existed")
	}
	content := twins
	var err error
	if content == nil {
		klog.Errorf("Update twin of device %s error, key:twin does not exist", deviceID)
		err = dttype.ErrorUpdate
		dealUpdateResult(context, deviceID, err, result)
		return err
	}

	inputTwins := make([]*v1alpha2.Twin, len(twins))
	for key, value := range twins {
		inputTwins[key] = value.DeepCopy()
	}

	// get added/deleted/updated twins
	dealTwinResult := DealMsgTwin(context, deviceID, inputTwins, dealType)
	add, deletes, update := dealTwinResult.Add, dealTwinResult.Delete, dealTwinResult.Update

	if dealType == RestDealType && dealTwinResult.Err != nil {
		SyncDeviceFromSqlite(context, deviceID)
		err = dealTwinResult.Err
		return err
	}

	if len(add) != 0 || len(deletes) != 0 || len(update) != 0 {
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			err = dtclient.DeviceTwinTrans(add, deletes, update)
			if err == nil {
				klog.Infof("insert succeed")
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			SyncDeviceFromSqlite(context, deviceID)
			klog.Errorf("Update device twin failed due to writing sql error: %v", err)
		}
	}

	klog.Infof("Finish update database")

	if err != nil && dealType == RestDealType {
		return err
	}

	if dealType == SyncDealType {
		delta, ok := dttype.BuildDeviceTwinDelta(device.Status.Twins)
		klog.Errorf("begin to deal ok %v : delta %v", ok, string(delta))
		if ok {
			dealDelta(context, deviceID, delta)
		}
	}

	if len(dealTwinResult.SyncResult.Status.Twins) > 0 && dealType == RestDealType {
		dealSyncResult(context, deviceID, dealTwinResult.SyncResult)
		klog.Errorf("sync result is %v", dealTwinResult.SyncResult.Status.Twins)
	}

	return nil
}

//dealUpdateResult build update result and send result, if success send the current state
func dealUpdateResult(context *dtcontext.DTContext, deviceID string, err error, payload []byte) error {
	klog.Infof("Deal update result of device %s: Build and send result", deviceID)
	// TODO: this topic mapper don't use, can be deleted, and should contain error code and error message
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETUpdateResultSuffix

	result := []byte("")
	if err == nil {
		result = payload
	} else {
		result = []byte("")
	}
	klog.Infof("Deal update result of device %s: send result", deviceID)
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
}

// dealDelta send delta
func dealDelta(context *dtcontext.DTContext, deviceID string, payload []byte) error {
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETDeltaSuffix
	// this topic will be sent to MQTT, device side will subscribe this topic to update device twins
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, payload))
}

// dealSyncResult build and send sync result, is delta update
func dealSyncResult(context *dtcontext.DTContext, deviceID string, device v1alpha2.Device) error {
	klog.Infof("Deal sync result of device %s: sync with cloud", deviceID)
	// this topic will be sent to Cloud side to sync reported twins，upstream controller will process this message
	resource := "device/" + deviceID + "/twin/edge_updated"
	return context.Send("",
		dtcommon.SendToCloud,
		dtcommon.CommModule,
		context.BuildModelMessage("resource", "", resource, model.UpdateOperation, device))
}

// DealGetTwin deal get twin event
func DealGetTwin(context *dtcontext.DTContext, deviceID string, payload []byte) error {
	klog.Infof("Deal the event of getting device %v twin", deviceID)
	msg := []byte("")
	device := v1alpha2.Device{}
	// TODO：this topic also use K8s CRD device structure, but how to contain error code or error message
	// temporarily, if error happens, just send empty []byte
	err := json.Unmarshal(payload, &device)
	if err != nil {
		klog.Errorf("Unmarshal twin info %s failed , err: %#v", string(payload), err)
		msg = []byte("")
	} else {
		doc, exist := context.GetDevice(deviceID)
		if !exist {
			klog.Errorf("Device %s not found while getting twin", deviceID)
			msg = []byte("")
		} else {
			var err error
			msg, err = dttype.BuildDeviceTwinResult(deviceID, doc.Status.Twins, RestDealType)
			if err != nil {
				klog.Errorf("Build state while deal get twin err: %#v", err)
				msg = []byte("")
			}
		}
	}
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETGetResultSuffix
	klog.Infof("Deal the event of getting twin of device %s: send result ", deviceID)
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, msg))
}

//dealtype 0:update ,2:cloud_update,1:detail result,3:deleted
func dealVersion(version *dttype.TwinVersion, reqVesion *dttype.TwinVersion, dealType int) (bool, error) {
	if dealType == RestDealType {
		version.EdgeVersion = version.EdgeVersion + 1
	} else if dealType >= SyncDealType {
		if reqVesion == nil {
			if dealType == SyncTwinDeleteDealType {
				return true, nil
			}
			return false, errors.New("Version not allowed be nil while syncing")
		}
		if version.CloudVersion > reqVesion.CloudVersion {
			return false, errors.New("Version not allowed")
		}
		if version.EdgeVersion > reqVesion.EdgeVersion {
			return false, errors.New("Not allowed to sync due to version conflict")
		}
		version.CloudVersion = reqVesion.CloudVersion
		version.EdgeVersion = reqVesion.EdgeVersion
	}
	return true, nil
}

func deleteTwinFromTwins(propertyName string, twins *[]v1alpha2.Twin) {
	for index, twin := range *twins {
		if twin.PropertyName == propertyName {
			*twins = append((*twins)[:index], (*twins)[index+1:]...)
			return
		}
	}
}

func dealTwinDelete(returnResult *dttype.DealTwinResult, deviceID string, cacheTwin *[]v1alpha2.Twin, msgTwin *v1alpha2.Twin, dealType int) error {
	deletedTwin := v1alpha2.Twin{
		PropertyName: msgTwin.PropertyName,
	}

	// delete from local cache
	deleteTwinFromTwins(msgTwin.PropertyName, cacheTwin)

	s := strings.Split(deviceID, "/")
	delete := dtclient.DeviceTwinPrimaryKey{
		PropertyName:    msgTwin.PropertyName,
		DeviceNamespace: s[0],
		DeviceName:      s[1],
	}
	returnResult.Delete = append(returnResult.Delete, delete)

	if dealType == RestDealType {
		returnResult.SyncResult.Status.Twins = append(returnResult.SyncResult.Status.Twins, deletedTwin)
	}
	return nil
}

func dealTwinCompare(returnResult *dttype.DealTwinResult, deviceID string, cacheTwin *[]v1alpha2.Twin, msgTwin *v1alpha2.Twin, dealType int) error {
	klog.Info("deal twin compare")
	if msgTwin == nil {
		return nil
	}
	if reflect.DeepEqual(msgTwin.Desired, v1alpha2.TwinProperty{}) && reflect.DeepEqual(msgTwin.Reported, v1alpha2.TwinProperty{}) {
		return nil
	}

	isExist := false
	index := 0
	cachedTwin := v1alpha2.Twin{}
	for k, twin := range *cacheTwin {
		if twin.PropertyName == msgTwin.PropertyName {
			isExist = true
			index = k
			cachedTwin = twin
			break
		}
	}
	if !isExist {
		return fmt.Errorf("device %s property %s not found", deviceID, msgTwin.PropertyName)
	}

	isChange := !reflect.DeepEqual(*msgTwin, cachedTwin)

	updatedTwin := v1alpha2.Twin{
		PropertyName: msgTwin.PropertyName,
		Desired:      cachedTwin.Desired,
		Reported:     cachedTwin.Reported,
	}
	s := strings.Split(deviceID, "/")
	deviceTwinUpdate := dtclient.DeviceTwinUpdate{
		DeviceName:      s[1],
		DeviceNamespace: s[0],
		PropertyName:    msgTwin.PropertyName,
		Cols:            make(map[string]interface{}),
	}

	if !isChange {
		return nil
	}

	if dealType != RestDealType {
		if !reflect.DeepEqual(msgTwin.Desired, v1alpha2.TwinProperty{}) && !reflect.DeepEqual(msgTwin.Desired, cachedTwin.Desired) {
			if msgTwin.Desired.Value != "" {
				updatedTwin.Desired.Value = msgTwin.Desired.Value
				deviceTwinUpdate.Cols["expected"] = msgTwin.Desired.Value
			}
			if msgTwin.Desired.Metadata != nil {
				updatedTwin.Desired.Metadata = msgTwin.Desired.Metadata
				deviceTwinUpdate.Cols["expected_meta"], _ = json.Marshal(msgTwin.Desired.Metadata)
			}
		}
	}

	if dealType == RestDealType {
		if !reflect.DeepEqual(msgTwin.Reported, v1alpha2.TwinProperty{}) && !reflect.DeepEqual(msgTwin.Reported, cachedTwin.Reported) {
			if msgTwin.Reported.Value != "" {
				updatedTwin.Reported.Value = msgTwin.Reported.Value
				deviceTwinUpdate.Cols["actual"] = msgTwin.Reported.Value
			}
			if msgTwin.Reported.Metadata != nil {
				updatedTwin.Reported.Metadata = msgTwin.Reported.Metadata
				deviceTwinUpdate.Cols["actual_meta"], _ = json.Marshal(msgTwin.Reported.Metadata)
			}
		}
	}

	// syncResult will sent to cloud to sync
	if dealType == RestDealType {
		returnResult.SyncResult.Status.Twins = append(returnResult.SyncResult.Status.Twins, updatedTwin)
	}

	// update local cache
	(*cacheTwin)[index] = updatedTwin

	if len(deviceTwinUpdate.Cols) != 0 {
		returnResult.Update = append(returnResult.Update, deviceTwinUpdate)
	}

	return nil
}

func dealTwinAdd(returnResult *dttype.DealTwinResult, deviceID string, cacheTwin *[]v1alpha2.Twin, msgTwin *v1alpha2.Twin, dealType int) error {
	if msgTwin == nil {
		return errors.New("The request body is wrong")
	}

	if reflect.DeepEqual(msgTwin.Desired, v1alpha2.TwinProperty{}) && reflect.DeepEqual(msgTwin.Desired, v1alpha2.TwinProperty{}) {
		return nil
	}

	newAddedTwin := v1alpha2.Twin{
		PropertyName: msgTwin.PropertyName,
		Desired:      msgTwin.Desired,
		Reported:     msgTwin.Reported,
	}

	// update local cache
	*cacheTwin = append(*cacheTwin, newAddedTwin)

	s := strings.Split(deviceID, "/")
	deviceTwin := dtclient.DeviceTwin{
		DeviceName:      s[1],
		DeviceNamespace: s[0],
		PropertyName:    msgTwin.PropertyName,
		Expected:        msgTwin.Desired.Value,
		Actual:          msgTwin.Reported.Value,
	}
	desiredMeta, _ := dtclient.ConvertMetaMapToString(msgTwin.Desired.Metadata)
	deviceTwin.ExpectedMeta = desiredMeta
	reportedMeta, _ := dtclient.ConvertMetaMapToString(msgTwin.Reported.Metadata)
	deviceTwin.ActualMeta = reportedMeta
	returnResult.Add = append(returnResult.Add, deviceTwin)

	if dealType == RestDealType {
		returnResult.SyncResult.Status.Twins = append(returnResult.SyncResult.Status.Twins, *msgTwin)
	}

	return nil
}

//DealMsgTwin get diff while updating twin, get added/deleted/updated
func DealMsgTwin(context *dtcontext.DTContext, deviceID string, msgTwins []*v1alpha2.Twin, dealType int) dttype.DealTwinResult {
	add := make([]dtclient.DeviceTwin, 0)
	deletes := make([]dtclient.DeviceTwinPrimaryKey, 0)
	update := make([]dtclient.DeviceTwinUpdate, 0)
	s := strings.Split(deviceID, "/")
	syncResult := v1alpha2.Device{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s[0],
			Name:      s[1],
		},
	}
	returnResult := dttype.DealTwinResult{
		Add:        add,
		Delete:     deletes,
		Update:     update,
		SyncResult: syncResult,
		Err:        nil,
	}

	device, ok := context.GetDevice(deviceID)
	if !ok {
		klog.Errorf("invalid device id: %v", deviceID)
		return dttype.DealTwinResult{Add: add,
			Delete:     deletes,
			Update:     update,
			SyncResult: syncResult,
			Err:        errors.New("invalid device id")}
	}

	if device.Status.Twins == nil {
		device.Status.Twins = make([]v1alpha2.Twin, 0)
	}
	twins := &device.Status.Twins

	var err error

	for _, msgTwin := range msgTwins {
		if msgTwin == nil {
			continue
		}
		if isExist, _ := isExist(msgTwin.PropertyName, twins); isExist {
			if dealType >= SyncDealType && msgTwin != nil && (msgTwin.Desired.Metadata["type"] == "") {
				klog.Infof("Not found metadata of twin")
			}
			if dealType >= SyncDealType && strings.Compare(msgTwin.Desired.Metadata["type"], "deleted") == 0 {
				err = dealTwinDelete(&returnResult, deviceID, twins, msgTwin, dealType)
				if err != nil {
					return returnResult
				}
				continue
			}
			err = dealTwinCompare(&returnResult, deviceID, twins, msgTwin, dealType)
			if err != nil {
				return returnResult
			}
		} else {
			err = dealTwinAdd(&returnResult, deviceID, twins, msgTwin, dealType)
			if err != nil {
				return returnResult
			}
		}
	}
	context.DeviceList.Store(deviceID, device)
	return returnResult
}

func isExist(propertyName string, twins *[]v1alpha2.Twin) (bool, *v1alpha2.Twin) {
	for _, value := range *twins {
		if value.PropertyName == propertyName {
			return true, &value
		}
	}
	return false, nil
}
