package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

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
	memActionCallBack[dtcommon.MemAdded] = dealMembershipAdd
	memActionCallBack[dtcommon.MemDeleted] = dealMembershipDelete
}

func dealMembershipAdd(context *dtcontext.DTContext, identity string, msg interface{}) (interface{}, error) {
	fmt.Printf("receive add device message")
	klog.Infof("Membership event")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	contentData, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("assertion failed")
	}

	addedDevice := &v1alpha2.Device{}
	err := json.Unmarshal(contentData, addedDevice)
	if err != nil {
		klog.Errorf("Unmarshal device info failed, err is %v", err)
		return nil, err
	}

	if !reflect.DeepEqual(addedDevice, &v1alpha2.Device{}) {
		addDevice(context, *addedDevice)
	}

	return nil, nil
}

func dealMembershipDelete(context *dtcontext.DTContext, identity string, msg interface{}) (interface{}, error) {
	klog.Infof("Membership event delete, device is %s", identity)
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	contentData, ok := message.Content.([]byte)
	if !ok {
		return nil, errors.New("assertion failed")
	}

	deletedDevice := v1alpha2.Device{}
	err := json.Unmarshal(contentData, &deletedDevice)
	if err != nil {
		klog.Errorf("Unmarshal deviceTransmitMsg info failed, err is %v", err)
		return nil, err
	}

	if !reflect.DeepEqual(deletedDevice, v1alpha2.Device{}) {
		removeDevice(context, deletedDevice)
	}
	return nil, nil
}

func dealMembershipGet(context *dtcontext.DTContext, identity string, msg interface{}) (interface{}, error) {
	klog.Infof("MEMBERSHIP EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}

	_, ok = message.Content.([]byte)
	if !ok {
		return nil, errors.New("assertion failed")
	}

	dealMembershipGetInner(context)
	return nil, nil
}

func addDevice(context *dtcontext.DTContext, device v1alpha2.Device) {
	klog.Infof("Add devices to edge group")
	dealType := SyncDealType
	fmt.Printf("############################## device is %v", device)

	uniqueKey := dtcommon.GenerateDeviceID(&device)
	_, isDeviceExist := context.GetDevice(uniqueKey)
	if isDeviceExist {
		DealDeviceTwin(context, uniqueKey, device.Status.Twins, dealType)
		//todo sync twin
		return
	}

	var deviceMutex sync.Mutex
	context.DeviceMutex.Store(uniqueKey, &deviceMutex)

	deviceStore := v1alpha2.Device{}
	deviceStore.Name = device.Name
	deviceStore.Namespace = device.Namespace
	context.DeviceList.Store(uniqueKey, &deviceStore)

	add := dtclient.ConvertCloudDeviceToTableDevice(device)

	var err error
	for i := 1; i <= dtcommon.RetryTimes; i++ {
		err = dtclient.SaveDevice(&add)
		if err == nil {
			break
		}
		time.Sleep(dtcommon.RetryInterval)
	}

	if err != nil {
		klog.Errorf("Add device %s failed due to some error ,err: %#v", uniqueKey, err)
		context.DeviceList.Delete(uniqueKey)
		context.Unlock(uniqueKey)
		return
		//todo
	}
	if device.Status.Twins != nil {
		klog.Infof("Add device twin during first adding device %s", uniqueKey)
		DealDeviceTwin(context, uniqueKey, device.Status.Twins, dealType)
	}

	// use membership/added, distinguish from removeDevice
	topic := dtcommon.MemETPrefix + context.NodeName + dtcommon.MemETAddSuffix
	result, err := json.Marshal(device)
	if err != nil {
		klog.Errorf("Marshal device failed, err is %v", err)
	} else {
		context.Send("",
			dtcommon.SendToEdge,
			dtcommon.CommModule,
			context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
	}
}

// removeDevice remove device from the edge group
func removeDevice(context *dtcontext.DTContext, toRemove v1alpha2.Device) {
	klog.Infof("Begin to remove devices")

	uniqueKey := dtcommon.GenerateDeviceID(&toRemove)
	//update sqlite
	_, deviceExist := context.GetDevice(uniqueKey)
	if !deviceExist {
		klog.Errorf("Remove device %s failed, not existed", uniqueKey)
		return
	}

	deletes := make([]dtclient.DevicePrimaryKey, 0)
	primaryKey := dtclient.DevicePrimaryKey{
		Namespace: toRemove.Namespace,
		Name:      toRemove.Name,
	}
	deletes = append(deletes, primaryKey)
	for i := 1; i <= dtcommon.RetryTimes; i++ {
		err := dtclient.DeleteDeviceTrans(deletes)
		if err != nil {
			klog.Errorf("Delete device %s failed at %d time, err: %#v", uniqueKey, i, err)
		} else {
			klog.Infof("Delete device %s successful", uniqueKey)
			break
		}
		time.Sleep(dtcommon.RetryInterval)
	}
	//todo
	context.DeviceList.Delete(uniqueKey)
	context.DeviceMutex.Delete(uniqueKey)

	// use membership/deleted, distinguish from addDevice
	topic := dtcommon.MemETPrefix + context.NodeName + dtcommon.MemETDeleteSuffix
	result, err := json.Marshal(toRemove)
	if err != nil {
		klog.Errorf("Marshal device failed, err is %v", err)
		return
	}

	context.Send("",
		dtcommon.SendToEdge,
		dtcommon.CommModule,
		context.BuildModelMessage(modules.BusGroup, "", topic, messagepkg.OperationPublish, result))
	klog.Infof("Remove device %s successful", uniqueKey)
}

// dealMembershipGetInner deal get membership event
func dealMembershipGetInner(context *dtcontext.DTContext) error {
	klog.Info("Deal getting membership event")
	result := []byte("")

	var devices []*v1alpha2.Device
	context.DeviceList.Range(func(key interface{}, value interface{}) bool {
		device, ok := value.(*v1alpha2.Device)
		if !ok {

		} else {
			devices = append(devices, device)
		}
		return true
	})

	result, err := dttype.BuildMembershipGetResult(devices)
	if err != nil {
		klog.Errorf("Marshal membership failed while deal get membership ,err: %#v", err)
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

	s := strings.Split(deviceID, "/")
	primaryKey := dtclient.DevicePrimaryKey{
		Name:      s[1],
		Namespace: s[0],
	}
	devices, err := dtclient.QueryDeviceByKey(primaryKey)
	if err != nil {
		klog.Errorf("query device attr failed: %v", err)
		return err
	}
	if len(*devices) == 0 {
		return errors.New("Not found device")
	}

	deviceTwinPrimaryKey := dtclient.DeviceTwinPrimaryKey{
		DeviceName:      s[1],
		DeviceNamespace: s[0],
	}
	deviceTwin, err := dtclient.QueryDeviceTwin(&deviceTwinPrimaryKey)
	if err != nil {
		klog.Errorf("query device twin failed: %v", err)
		return err
	}

	// convert from table device structure to K8s device structure
	deviceCache := dtclient.GetK8sDeviceFromDeviceTwin(*deviceTwin)

	uniqueKey := dtcommon.GenerateDeviceID(deviceCache)
	context.DeviceList.Store(uniqueKey, deviceCache)

	return nil
}
