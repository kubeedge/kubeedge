package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
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
	//DealExpected deal exepected
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

func dealTwinSync(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.Infof("Twin Sync EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}
	result := []byte("")
	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invalid message content")
	}

	msgTwin, err := dttype.UnmarshalDeviceTwinUpdate(content)
	if err != nil {
		klog.Errorf("Unmarshal update request body failed, err: %#v", err)
		dealUpdateResult(context, "", "", dtcommon.BadRequestCode, errors.New("Unmarshal update request body failed, Please check the request"), result)
		return nil, err
	}

	klog.Infof("Begin to update twin of the device %s", resource)
	eventID := msgTwin.EventID
	context.Lock(resource)
	DealDeviceTwin(context, resource, eventID, msgTwin.Twin, SyncDealType)
	context.Unlock(resource)
	//todo send ack
	return nil, nil
}

func dealTwinGet(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.Infof("Twin Get EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invalid message content")
	}

	DealGetTwin(context, resource, content)
	return nil, nil
}

func dealTwinUpdate(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.Infof("Twin Update EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	content, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("invalid message content")
	}

	context.Lock(resource)
	Updated(context, resource, content)
	context.Unlock(resource)
	return nil, nil
}

// Updated update the snapshot
func Updated(context *dtcontext.DTContext, deviceID string, payload []byte) {
	result := []byte("")
	msg, err := dttype.UnmarshalDeviceTwinUpdate(payload)
	if err != nil {
		klog.Errorf("Unmarshal update request body failed, err: %#v", err)
		dealUpdateResult(context, "", "", dtcommon.BadRequestCode, err, result)
		return
	}
	klog.Infof("Begin to update twin of the device %s", deviceID)
	eventID := msg.EventID
	DealDeviceTwin(context, deviceID, eventID, msg.Twin, RestDealType)
}

//DealDeviceTwin deal device twin
func DealDeviceTwin(context *dtcontext.DTContext, deviceID string, eventID string, msgTwin map[string]*dttype.MsgTwin, dealType int) error {
	klog.Infof("Begin to deal device twin of the device %s", deviceID)
	now := time.Now().UnixNano() / 1e6
	result := []byte("")
	deviceModel, isExist := context.GetDevice(deviceID)
	if !isExist {
		klog.Errorf("Update twin rejected due to the device %s is not existed", deviceID)
		dealUpdateResult(context, deviceID, eventID, dtcommon.NotFoundCode, errors.New("Update rejected due to the device is not existed"), result)
		return errors.New("Update rejected due to the device is not existed")
	}
	content := msgTwin
	var err error
	if content == nil {
		klog.Errorf("Update twin of device %s error, the update request body not have key:twin", deviceID)
		err = errors.New("Update twin error, the update request body not have key:twin")
		dealUpdateResult(context, deviceID, eventID, dtcommon.BadRequestCode, err, result)
		return err
	}
	dealTwinResult := DealMsgTwin(context, deviceID, content, dealType)

	add, deletes, update := dealTwinResult.Add, dealTwinResult.Delete, dealTwinResult.Update
	if dealType == RestDealType && dealTwinResult.Err != nil {
		SyncDeviceFromSqlite(context, deviceID)
		err = dealTwinResult.Err
		updateResult, _ := dttype.BuildDeviceTwinResult(dttype.BaseMessage{EventID: eventID, Timestamp: now}, dealTwinResult.Result, 0)
		dealUpdateResult(context, deviceID, eventID, dtcommon.BadRequestCode, err, updateResult)
		return err
	}
	if len(add) != 0 || len(deletes) != 0 || len(update) != 0 {
		for i := 1; i <= dtcommon.RetryTimes; i++ {
			err = dtclient.DeviceTwinTrans(add, deletes, update)
			if err == nil {
				break
			}
			time.Sleep(dtcommon.RetryInterval)
		}
		if err != nil {
			SyncDeviceFromSqlite(context, deviceID)
			klog.Errorf("Update device twin failed due to writing sql error: %v", err)
		}
	}

	if err != nil && dealType == RestDealType {
		updateResult, _ := dttype.BuildDeviceTwinResult(dttype.BaseMessage{EventID: eventID, Timestamp: now}, dealTwinResult.Result, dealType)
		dealUpdateResult(context, deviceID, eventID, dtcommon.InternalErrorCode, err, updateResult)
		return err
	}
	if dealType == RestDealType {
		updateResult, _ := dttype.BuildDeviceTwinResult(dttype.BaseMessage{EventID: eventID, Timestamp: now}, dealTwinResult.Result, dealType)
		dealUpdateResult(context, deviceID, eventID, dtcommon.InternalErrorCode, nil, updateResult)
	}
	if len(dealTwinResult.Document) > 0 {
		dealDocument(context, deviceID, dttype.BaseMessage{EventID: eventID, Timestamp: now}, dealTwinResult.Document)
	}

	delta, ok := dttype.BuildDeviceTwinDelta(dttype.BuildBaseMessage(), deviceModel.Twin)
	if ok {
		dealDelta(context, deviceID, delta)
	}

	if len(dealTwinResult.SyncResult) > 0 {
		dealSyncResult(context, deviceID, dttype.BuildBaseMessage(), dealTwinResult.SyncResult)
	}
	return nil
}

//dealUpdateResult build update result and send result, if success send the current state
func dealUpdateResult(context *dtcontext.DTContext, deviceID string, eventID string, code int, err error, payload []byte) error {
	klog.Infof("Deal update result of device %s: Build and send result", deviceID)

	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETUpdateResultSuffix
	reason := ""
	para := dttype.Parameter{
		EventID: eventID,
		Code:    code,
		Reason:  reason}
	result := []byte("")
	var jsonErr error
	if err == nil {
		result = payload
	} else {
		para.Reason = err.Error()
		result, jsonErr = dttype.BuildErrorResult(para)
		if jsonErr != nil {
			klog.Errorf("Unmarshal error result of device %s error, err: %v", deviceID, jsonErr)
			return jsonErr
		}
	}
	klog.Infof("Deal update result of device %s: send result", deviceID)
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, "publish", result))
}

// dealDelta  send delta
func dealDelta(context *dtcontext.DTContext, deviceID string, payload []byte) error {
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETDeltaSuffix
	klog.Infof("Deal delta of device %s: send delta", deviceID)
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, "publish", payload))
}

// dealSyncResult build and send sync result, is delta update
func dealSyncResult(context *dtcontext.DTContext, deviceID string, baseMessage dttype.BaseMessage, twin map[string]*dttype.MsgTwin) error {
	klog.Infof("Deal sync result of device %s: sync with cloud", deviceID)
	resource := "device/" + deviceID + "/twin/edge_updated"
	return context.Send("",
		dtcommon.SendToCloud,
		dtcommon.CommModule,
		context.BuildModelMessage("resource", "", resource, "update", dttype.DeviceTwinResult{BaseMessage: baseMessage, Twin: twin}))
}

//dealDocument build document and save current state as last state, update sqlite
func dealDocument(context *dtcontext.DTContext, deviceID string, baseMessage dttype.BaseMessage, twinDocument map[string]*dttype.TwinDoc) error {
	klog.Infof("Deal document of device %s: build and send document", deviceID)
	payload, _ := dttype.BuildDeviceTwinDocument(baseMessage, twinDocument)
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETDocumentSuffix
	klog.Infof("Deal document of device %s: send document", deviceID)
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, "publish", payload))
}

// DealGetTwin deal get twin event
func DealGetTwin(context *dtcontext.DTContext, deviceID string, payload []byte) error {
	klog.Info("Deal the event of getting twin")
	msg := []byte("")
	para := dttype.Parameter{}
	edgeGet, err := dttype.UnmarshalBaseMessage(payload)
	if err != nil {
		klog.Errorf("Unmarshal twin info %s failed , err: %#v", string(payload), err)
		para.Code = dtcommon.BadRequestCode
		para.Reason = fmt.Sprintf("Unmarshal twin info %s failed , err: %#v", string(payload), err)
		var jsonErr error
		msg, jsonErr = dttype.BuildErrorResult(para)
		if jsonErr != nil {
			klog.Errorf("Unmarshal error result error, err: %v", jsonErr)
			return jsonErr
		}
	} else {
		para.EventID = edgeGet.EventID
		doc, exist := context.GetDevice(deviceID)
		if !exist {
			klog.Errorf("Device %s not found while getting twin", deviceID)
			para.Code = dtcommon.NotFoundCode
			para.Reason = fmt.Sprintf("Device %s not found while getting twin", deviceID)
			var jsonErr error
			msg, jsonErr = dttype.BuildErrorResult(para)
			if jsonErr != nil {
				klog.Errorf("Unmarshal error result error, err: %v", jsonErr)
				return jsonErr
			}
		} else {
			now := time.Now().UnixNano() / 1e6
			var err error
			msg, err = dttype.BuildDeviceTwinResult(dttype.BaseMessage{EventID: edgeGet.EventID, Timestamp: now}, doc.Twin, RestDealType)
			if err != nil {
				klog.Errorf("Build state while deal get twin err: %#v", err)
				para.Code = dtcommon.InternalErrorCode
				para.Reason = fmt.Sprintf("Build state while deal get twin err: %#v", err)
				var jsonErr error
				msg, jsonErr = dttype.BuildErrorResult(para)
				if jsonErr != nil {
					klog.Errorf("Unmarshal error result error, err: %v", jsonErr)
					return jsonErr
				}
			}
		}
	}
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETGetResultSuffix
	klog.Infof("Deal the event of getting twin of device %s: send result ", deviceID)
	return context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, "publish", msg))
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

func dealTwinDelete(returnResult *dttype.DealTwinResult, deviceID string, key string, twin *dttype.MsgTwin, msgTwin *dttype.MsgTwin, dealType int) error {
	document := returnResult.Document
	document[key] = &dttype.TwinDoc{}
	copytwin := dttype.CopyMsgTwin(twin, true)
	document[key].LastState = &copytwin
	cols := make(map[string]interface{})
	syncResult := returnResult.SyncResult
	syncResult[key] = &dttype.MsgTwin{}
	update := returnResult.Update
	isChange := false
	if msgTwin == nil && dealType == RestDealType && *twin.Optional || dealType >= SyncDealType && strings.Compare(msgTwin.Metadata.Type, "deleted") == 0 {
		if twin.Metadata != nil && strings.Compare(twin.Metadata.Type, "deleted") == 0 {
			return nil
		}
		if dealType != RestDealType {
			dealType = SyncTwinDeleteDealType
		}
		hasTwinExpected := true
		if twin.ExpectedVersion == nil {
			twin.ExpectedVersion = &dttype.TwinVersion{}
			hasTwinExpected = false
		}
		if hasTwinExpected {
			expectedVersion := twin.ExpectedVersion

			var msgTwinExpectedVersion *dttype.TwinVersion
			if dealType != RestDealType {
				msgTwinExpectedVersion = msgTwin.ExpectedVersion
			}
			ok, _ := dealVersion(expectedVersion, msgTwinExpectedVersion, dealType)
			if !ok {
				if dealType != RestDealType {
					copySync := dttype.CopyMsgTwin(twin, false)
					syncResult[key] = &copySync
					delete(document, key)
					returnResult.SyncResult = syncResult
					return nil
				}
			} else {
				expectedVersionJSON, _ := json.Marshal(expectedVersion)
				cols["expected_version"] = string(expectedVersionJSON)
				cols["attr_type"] = "deleted"
				cols["expected_meta"] = nil
				cols["expected"] = nil
				if twin.Expected == nil {
					twin.Expected = &dttype.TwinValue{}
				}
				twin.Expected.Value = nil
				twin.Expected.Metadata = nil
				twin.ExpectedVersion = expectedVersion
				twin.Metadata = &dttype.TypeMetadata{Type: "deleted"}
				if dealType == RestDealType {
					copySync := dttype.CopyMsgTwin(twin, false)
					syncResult[key] = &copySync
				}
				document[key].CurrentState = nil
				isChange = true
			}
		}
		hasTwinActual := true

		if twin.ActualVersion == nil {
			twin.ActualVersion = &dttype.TwinVersion{}
			hasTwinActual = false
		}
		if hasTwinActual {
			actualVersion := twin.ActualVersion
			var msgTwinActualVersion *dttype.TwinVersion
			if dealType != RestDealType {
				msgTwinActualVersion = msgTwin.ActualVersion
			}
			ok, _ := dealVersion(actualVersion, msgTwinActualVersion, dealType)
			if !ok {
				if dealType != RestDealType {
					copySync := dttype.CopyMsgTwin(twin, false)
					syncResult[key] = &copySync
					delete(document, key)
					returnResult.SyncResult = syncResult
					return nil
				}
			} else {
				actualVersionJSON, _ := json.Marshal(actualVersion)

				cols["actual_version"] = string(actualVersionJSON)
				cols["attr_type"] = "deleted"
				cols["actual_meta"] = nil
				cols["actual"] = nil
				if twin.Actual == nil {
					twin.Actual = &dttype.TwinValue{}
				}
				twin.Actual.Value = nil
				twin.Actual.Metadata = nil
				twin.ActualVersion = actualVersion
				twin.Metadata = &dttype.TypeMetadata{Type: "deleted"}
				if dealType == RestDealType {
					copySync := dttype.CopyMsgTwin(twin, false)
					syncResult[key] = &copySync
				}
				document[key].CurrentState = nil
				isChange = true
			}
		}
	}

	if isChange {
		update = append(update, dtclient.DeviceTwinUpdate{DeviceID: deviceID, Name: key, Cols: cols})
		returnResult.Update = update
		if dealType == RestDealType {
			returnResult.Result[key] = nil
			returnResult.SyncResult = syncResult
		} else {
			delete(syncResult, key)
		}
		returnResult.Document = document
	} else {
		delete(document, key)
		delete(syncResult, key)
	}

	return nil
}

//0:expected ,1 :actual
func isTwinValueDiff(twin *dttype.MsgTwin, msgTwin *dttype.MsgTwin, dealType int) (bool, error) {
	hasTwin := false
	hasMsgTwin := false
	twinValue := twin.Expected
	msgTwinValue := msgTwin.Expected

	if dealType == DealActual {
		twinValue = twin.Actual
		msgTwinValue = msgTwin.Actual
	}
	if twinValue != nil {
		hasTwin = true
	}
	if msgTwinValue != nil {
		hasMsgTwin = true
	}
	valueType := stringType
	if strings.Compare(twin.Metadata.Type, "deleted") == 0 {
		if msgTwin.Metadata != nil {
			valueType = msgTwin.Metadata.Type
		}
	} else {
		valueType = twin.Metadata.Type
	}
	if hasMsgTwin {
		if hasTwin {
			err := dtcommon.ValidateValue(valueType, *msgTwinValue.Value)
			if err != nil {
				return false, err
			}
			return true, nil
		}
		return true, nil
	}
	return false, nil
}

func dealTwinCompare(returnResult *dttype.DealTwinResult, deviceID string, key string, twin *dttype.MsgTwin, msgTwin *dttype.MsgTwin, dealType int) error {
	klog.Info("dealtwincompare")
	now := time.Now().UnixNano() / 1e6

	document := returnResult.Document
	document[key] = &dttype.TwinDoc{}
	copytwin := dttype.CopyMsgTwin(twin, true)
	document[key].LastState = &copytwin
	if strings.Compare(twin.Metadata.Type, "deleted") == 0 {
		document[key].LastState = nil
	}

	cols := make(map[string]interface{})

	syncResult := returnResult.SyncResult
	syncResult[key] = &dttype.MsgTwin{}
	update := returnResult.Update
	isChange := false
	isSyncAllow := true
	if msgTwin == nil {
		return nil
	}
	expectedOk, expectedErr := isTwinValueDiff(twin, msgTwin, DealExpected)
	if expectedOk {
		value := msgTwin.Expected.Value
		meta := dttype.ValueMetadata{Timestamp: now}
		if twin.ExpectedVersion == nil {
			twin.ExpectedVersion = &dttype.TwinVersion{}
		}
		version := twin.ExpectedVersion
		var msgTwinExpectedVersion *dttype.TwinVersion
		if dealType != RestDealType {
			msgTwinExpectedVersion = msgTwin.ExpectedVersion
		}
		ok, err := dealVersion(version, msgTwinExpectedVersion, dealType)
		if !ok {
			// if reject the sync,  set the syncResult and then send the edge_updated msg
			if dealType != RestDealType {
				syncResult[key].Expected = &dttype.TwinValue{Value: twin.Expected.Value, Metadata: twin.Expected.Metadata}
				syncResult[key].ExpectedVersion = &dttype.TwinVersion{CloudVersion: twin.ExpectedVersion.CloudVersion, EdgeVersion: twin.ExpectedVersion.EdgeVersion}

				syncOptional := *twin.Optional
				syncResult[key].Optional = &syncOptional

				metaJSON, _ := json.Marshal(twin.Metadata)
				var meta dttype.TypeMetadata
				json.Unmarshal(metaJSON, &meta)
				syncResult[key].Metadata = &meta

				isSyncAllow = false
			} else {
				returnResult.Err = err
				return err
			}
		} else {
			metaJSON, _ := json.Marshal(meta)
			versionJSON, _ := json.Marshal(version)
			cols["expected"] = value
			cols["expected_meta"] = string(metaJSON)
			cols["expected_version"] = string(versionJSON)
			if twin.Expected == nil {
				twin.Expected = &dttype.TwinValue{}
			}
			twin.Expected.Value = value
			twin.Expected.Metadata = &meta
			twin.ExpectedVersion = version
			// if rest update, set the syncResult and send the edge_updated msg
			if dealType == RestDealType {
				syncResult[key].Expected = &dttype.TwinValue{Value: value, Metadata: &meta}
				syncResult[key].ExpectedVersion = &dttype.TwinVersion{CloudVersion: version.CloudVersion, EdgeVersion: version.EdgeVersion}
				syncOptional := *twin.Optional
				syncResult[key].Optional = &syncOptional
				metaJSON, _ := json.Marshal(twin.Metadata)
				var meta dttype.TypeMetadata
				json.Unmarshal(metaJSON, &meta)
				syncResult[key].Metadata = &meta
			}
			isChange = true
		}
	} else {
		if expectedErr != nil && dealType == RestDealType {
			returnResult.Err = expectedErr
			return expectedErr
		}
	}
	actualOk, actualErr := isTwinValueDiff(twin, msgTwin, DealActual)
	if actualOk && isSyncAllow {
		value := msgTwin.Actual.Value
		meta := dttype.ValueMetadata{Timestamp: now}
		if twin.ActualVersion == nil {
			twin.ActualVersion = &dttype.TwinVersion{}
		}
		version := twin.ActualVersion
		var msgTwinActualVersion *dttype.TwinVersion
		if dealType != RestDealType {
			msgTwinActualVersion = msgTwin.ActualVersion
		}
		ok, err := dealVersion(version, msgTwinActualVersion, dealType)
		if !ok {
			if dealType != RestDealType {
				syncResult[key].Actual = &dttype.TwinValue{Value: twin.Actual.Value, Metadata: twin.Actual.Metadata}
				syncResult[key].ActualVersion = &dttype.TwinVersion{CloudVersion: twin.ActualVersion.CloudVersion, EdgeVersion: twin.ActualVersion.EdgeVersion}
				syncOptional := *twin.Optional
				syncResult[key].Optional = &syncOptional
				metaJSON, _ := json.Marshal(twin.Metadata)
				var meta dttype.TypeMetadata
				json.Unmarshal(metaJSON, &meta)
				syncResult[key].Metadata = &meta
				isSyncAllow = false
			} else {
				returnResult.Err = err
				return err
			}
		} else {
			metaJSON, _ := json.Marshal(meta)
			versionJSON, _ := json.Marshal(version)
			cols["actual"] = value
			cols["actual_meta"] = string(metaJSON)
			cols["actual_version"] = string(versionJSON)
			if twin.Actual == nil {
				twin.Actual = &dttype.TwinValue{}
			}
			twin.Actual.Value = value
			twin.Actual.Metadata = &meta
			twin.ActualVersion = version
			if dealType == RestDealType {
				syncResult[key].Actual = &dttype.TwinValue{Value: msgTwin.Actual.Value, Metadata: &meta}
				syncOptional := *twin.Optional
				syncResult[key].Optional = &syncOptional
				metaJSON, _ := json.Marshal(twin.Metadata)
				var meta dttype.TypeMetadata
				json.Unmarshal(metaJSON, &meta)
				syncResult[key].Metadata = &meta
				syncResult[key].ActualVersion = &dttype.TwinVersion{CloudVersion: version.CloudVersion, EdgeVersion: version.EdgeVersion}
			}
			isChange = true
		}
	} else {
		if actualErr != nil && dealType == RestDealType {
			returnResult.Err = actualErr
			return actualErr
		}
	}

	if isSyncAllow {
		if msgTwin.Optional != nil {
			if *msgTwin.Optional != *twin.Optional && *twin.Optional {
				optional := *msgTwin.Optional
				cols["optional"] = optional
				twin.Optional = &optional
				syncOptional := *twin.Optional
				syncResult[key].Optional = &syncOptional
				isChange = true
			}
		}
		// if update the deleted twin, allow to update attr_type
		if msgTwin.Metadata != nil {
			msgMetaJSON, _ := json.Marshal(msgTwin.Metadata)
			twinMetaJSON, _ := json.Marshal(twin.Metadata)
			if strings.Compare(string(msgMetaJSON), string(twinMetaJSON)) != 0 {
				meta := dttype.CopyMsgTwin(msgTwin, true)
				meta.Metadata.Type = ""
				metaJSON, _ := json.Marshal(meta.Metadata)
				cols["metadata"] = string(metaJSON)
				if strings.Compare(twin.Metadata.Type, "deleted") == 0 {
					cols["attr_type"] = msgTwin.Metadata.Type
					twin.Metadata.Type = msgTwin.Metadata.Type
					var meta dttype.TypeMetadata
					json.Unmarshal(msgMetaJSON, &meta)
					syncResult[key].Metadata = &meta
				}
				isChange = true
			}
		} else {
			if strings.Compare(twin.Metadata.Type, "deleted") == 0 {
				twin.Metadata = &dttype.TypeMetadata{Type: stringType}
				cols["attr_type"] = stringType
				syncResult[key].Metadata = twin.Metadata
				isChange = true
			}
		}
	}
	if isChange {
		update = append(update, dtclient.DeviceTwinUpdate{DeviceID: deviceID, Name: key, Cols: cols})
		returnResult.Update = update
		current := dttype.CopyMsgTwin(twin, true)
		document[key].CurrentState = &current
		returnResult.Document = document

		if dealType == RestDealType {
			copyResult := dttype.CopyMsgTwin(syncResult[key], true)
			returnResult.Result[key] = &copyResult
			returnResult.SyncResult = syncResult
		} else {
			if !isSyncAllow {
				returnResult.SyncResult = syncResult
			} else {
				delete(syncResult, key)
			}
		}
	} else {
		if dealType == RestDealType {
			delete(document, key)
			delete(syncResult, key)
		} else {
			delete(document, key)
			if !isSyncAllow {
				returnResult.SyncResult = syncResult
			} else {
				delete(syncResult, key)
			}
		}
	}
	return nil
}

func dealTwinAdd(returnResult *dttype.DealTwinResult, deviceID string, key string, twins map[string]*dttype.MsgTwin, msgTwin *dttype.MsgTwin, dealType int) error {
	now := time.Now().UnixNano() / 1e6
	document := returnResult.Document
	document[key] = &dttype.TwinDoc{}
	document[key].LastState = nil
	if msgTwin == nil {
		return errors.New("The request body is wrong")
	}
	deviceTwin := dttype.MsgTwinToDeviceTwin(key, msgTwin)
	deviceTwin.DeviceID = deviceID
	syncResult := returnResult.SyncResult
	syncResult[key] = &dttype.MsgTwin{}
	isChange := false
	//add deleted twin when syncing from cloud: add version
	if dealType != RestDealType && strings.Compare(msgTwin.Metadata.Type, "deleted") == 0 {
		if msgTwin.ExpectedVersion != nil {
			versionJSON, _ := json.Marshal(msgTwin.ExpectedVersion)
			deviceTwin.ExpectedVersion = string(versionJSON)
		}
		if msgTwin.ActualVersion != nil {
			versionJSON, _ := json.Marshal(msgTwin.ActualVersion)
			deviceTwin.ActualVersion = string(versionJSON)
		}
	}

	if msgTwin.Expected != nil {
		version := &dttype.TwinVersion{}
		var msgTwinExpectedVersion *dttype.TwinVersion
		if dealType != RestDealType {
			msgTwinExpectedVersion = msgTwin.ExpectedVersion
		}
		ok, err := dealVersion(version, msgTwinExpectedVersion, dealType)
		if !ok {
			// not match
			if dealType == RestDealType {
				returnResult.Err = err
				return err
			}
			// reject add twin
			return nil
		}
		// value type default string
		valueType := stringType
		if msgTwin.Metadata != nil {
			valueType = msgTwin.Metadata.Type
		}

		err = dtcommon.ValidateValue(valueType, *msgTwin.Expected.Value)
		if err == nil {
			meta := dttype.ValueMetadata{Timestamp: now}
			metaJSON, _ := json.Marshal(meta)
			versionJSON, _ := json.Marshal(version)
			deviceTwin.ExpectedMeta = string(metaJSON)
			deviceTwin.ExpectedVersion = string(versionJSON)
			deviceTwin.Expected = *msgTwin.Expected.Value
			isChange = true
		} else {
			delete(document, key)
			delete(syncResult, key)
			// reject add twin, if rest add return the err, while sync add return nil
			if dealType == RestDealType {
				returnResult.Err = err
				return err
			}
			return nil
		}
	}

	if msgTwin.Actual != nil {
		version := &dttype.TwinVersion{}
		var msgTwinActualVersion *dttype.TwinVersion
		if dealType != RestDealType {
			msgTwinActualVersion = msgTwin.ActualVersion
		}
		ok, err := dealVersion(version, msgTwinActualVersion, dealType)
		if !ok {
			if dealType == RestDealType {
				returnResult.Err = err
				return err
			}
			return nil
		}
		valueType := stringType
		if msgTwin.Metadata != nil {
			valueType = msgTwin.Metadata.Type
		}
		err = dtcommon.ValidateValue(valueType, *msgTwin.Actual.Value)
		if err == nil {
			meta := dttype.ValueMetadata{Timestamp: now}
			metaJSON, _ := json.Marshal(meta)
			versionJSON, _ := json.Marshal(version)
			deviceTwin.ActualMeta = string(metaJSON)
			deviceTwin.ActualVersion = string(versionJSON)
			deviceTwin.Actual = *msgTwin.Actual.Value
			isChange = true
		} else {
			delete(document, key)
			delete(syncResult, key)
			if dealType == RestDealType {
				returnResult.Err = err
				return err
			}
			return nil
		}
	}

	//add the optional of twin
	if msgTwin.Optional != nil {
		optional := *msgTwin.Optional
		deviceTwin.Optional = optional
		isChange = true
	} else {
		deviceTwin.Optional = true
		isChange = true
	}

	//add the metadata of the twin
	if msgTwin.Metadata != nil {
		//todo
		deviceTwin.AttrType = msgTwin.Metadata.Type
		msgTwin.Metadata.Type = ""
		metaJSON, _ := json.Marshal(msgTwin.Metadata)
		deviceTwin.Metadata = string(metaJSON)
		msgTwin.Metadata.Type = deviceTwin.AttrType
		isChange = true
	} else {
		deviceTwin.AttrType = stringType
		isChange = true
	}

	if isChange {
		twins[key] = dttype.DeviceTwinToMsgTwin([]dtclient.DeviceTwin{deviceTwin})[key]
		add := returnResult.Add
		add = append(add, deviceTwin)
		returnResult.Add = add

		copytwin := dttype.CopyMsgTwin(twins[key], true)
		if strings.Compare(twins[key].Metadata.Type, "deleted") == 0 {
			document[key].CurrentState = nil
		} else {
			document[key].CurrentState = &copytwin
		}
		returnResult.Document = document

		copySync := dttype.CopyMsgTwin(twins[key], false)
		syncResult[key] = &copySync
		if dealType == RestDealType {
			copyResult := dttype.CopyMsgTwin(syncResult[key], true)
			returnResult.Result[key] = &copyResult
			returnResult.SyncResult = syncResult
		} else {
			delete(syncResult, key)
		}
	} else {
		delete(document, key)
		delete(syncResult, key)
	}

	return nil
}

//DealMsgTwin get diff while updating twin
func DealMsgTwin(context *dtcontext.DTContext, deviceID string, msgTwins map[string]*dttype.MsgTwin, dealType int) dttype.DealTwinResult {
	add := make([]dtclient.DeviceTwin, 0)
	deletes := make([]dtclient.DeviceDelete, 0)
	update := make([]dtclient.DeviceTwinUpdate, 0)
	result := make(map[string]*dttype.MsgTwin)
	syncResult := make(map[string]*dttype.MsgTwin)
	document := make(map[string]*dttype.TwinDoc)
	returnResult := dttype.DealTwinResult{Add: add,
		Delete:     deletes,
		Update:     update,
		Result:     result,
		SyncResult: syncResult,
		Document:   document,
		Err:        nil}

	deviceModel, ok := context.GetDevice(deviceID)
	if !ok {
		klog.Errorf("invalid device id")
		return dttype.DealTwinResult{Add: add,
			Delete:     deletes,
			Update:     update,
			Result:     result,
			SyncResult: syncResult,
			Document:   document,
			Err:        errors.New("invalid device id")}
	}

	twins := deviceModel.Twin
	if twins == nil {
		deviceModel.Twin = make(map[string]*dttype.MsgTwin)
		twins = deviceModel.Twin
	}

	var err error
	for key, msgTwin := range msgTwins {
		if twin, exist := twins[key]; exist {
			if dealType >= 1 && msgTwin != nil && (msgTwin.Metadata == nil) {
				klog.Infof("Not found metadata of twin")
			}
			if msgTwin == nil && dealType == 0 || dealType >= 1 && strings.Compare(msgTwin.Metadata.Type, "deleted") == 0 {
				err = dealTwinDelete(&returnResult, deviceID, key, twin, msgTwin, dealType)
				if err != nil {
					return returnResult
				}
				continue
			}
			err = dealTwinCompare(&returnResult, deviceID, key, twin, msgTwin, dealType)
			if err != nil {
				return returnResult
			}
		} else {
			err = dealTwinAdd(&returnResult, deviceID, key, twins, msgTwin, dealType)
			if err != nil {
				return returnResult
			}
		}
	}
	context.DeviceList.Store(deviceID, deviceModel)
	return returnResult
}
