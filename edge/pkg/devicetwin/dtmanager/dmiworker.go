/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiserver"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
	pb "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1alpha1"
)

//TwinWorker deal twin event
type DMIWorker struct {
	Worker
	Group    string
	dmiCache *dmiserver.DMICache
	//dmiActionCallBack map for action to callback
	dmiActionCallBack map[string]CallBack
}

func (dw *DMIWorker) init() {
	dw.dmiCache = &dmiserver.DMICache{
		MapperMu:        &sync.Mutex{},
		DeviceMu:        &sync.Mutex{},
		DeviceModelMu:   &sync.Mutex{},
		MapperList:      make(map[string]*pb.MapperInfo),
		DeviceList:      make(map[string]*v1alpha2.Device),
		DeviceModelList: make(map[string]*v1alpha2.DeviceModel),
	}

	dw.initDMIActionCallBack()
	dw.initDeviceModelInfoFromDB()
	dw.initDeviceInfoFromDB()
	dw.initDeviceMapperInfoFromDB()
}

//Start worker
func (dw DMIWorker) Start() {
	klog.Infoln("dmi worker start")
	dw.init()

	go dmiserver.StartDMIServer(dw.dmiCache)

	for {
		select {
		case msg, ok := <-dw.ReceiverChan:
			if !ok {
				return
			}

			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := dw.dmiActionCallBack[dtMsg.Action]; exist {
					err := fn(dw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("DMIModule deal %s event failed: %v", dtMsg.Action, err)
					}
				} else {
					klog.Errorf("DMIModule deal %s event failed, not found callback", dtMsg.Action)
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

func (dw *DMIWorker) initDMIActionCallBack() {
	dw.dmiActionCallBack = make(map[string]CallBack)
	dw.dmiActionCallBack[dtcommon.MetaDeviceOperation] = dw.dealMetaDeviceOperation
}

func (dw *DMIWorker) dealMetaDeviceOperation(context *dtcontext.DTContext, resource string, msg interface{}) error {
	message, ok := msg.(*model.Message)
	if !ok {
		return errors.New("msg not Message type")
	}
	resources := strings.Split(message.Router.Resource, "/")
	if len(resources) != 3 {
		return fmt.Errorf("wrong resources %s", message.Router.Resource)
	}
	var device v1alpha2.Device
	var dm v1alpha2.DeviceModel
	switch resources[1] {
	case constants.ResourceTypeDevice:
		err := json.Unmarshal(message.Content.([]byte), &device)
		if err != nil {
			return fmt.Errorf("invalid message content with err: %+v", err)
		}
		switch message.GetOperation() {
		case model.InsertOperation:
			err = dmiclient.DMIClientsImp.RegisterDevice(&device)
			if err != nil {
				klog.Errorf("add device %s failed with err: %v", device.Name, err)
				return err
			}
			dw.dmiCache.DeviceMu.Lock()
			dw.dmiCache.DeviceList[device.Name] = &device
			dw.dmiCache.DeviceMu.Unlock()
		case model.DeleteOperation:
			err = dmiclient.DMIClientsImp.RemoveDevice(&device)
			if err != nil {
				klog.Errorf("delete device %s failed with err: %v", device.Name, err)
				return err
			}
			dw.dmiCache.DeviceMu.Lock()
			delete(dw.dmiCache.DeviceList, device.Name)
			dw.dmiCache.DeviceMu.Unlock()
		case model.UpdateOperation:
			err = dmiclient.DMIClientsImp.UpdateDevice(&device)
			if err != nil {
				klog.Errorf("udpate device %s failed with err: %v", device.Name, err)
				return err
			}
			dw.dmiCache.DeviceMu.Lock()
			dw.dmiCache.DeviceList[device.Name] = &device
			dw.dmiCache.DeviceMu.Unlock()
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}
	case constants.ResourceTypeDeviceModel:
		err := json.Unmarshal(message.Content.([]byte), &dm)
		if err != nil {
			return fmt.Errorf("invalid message content with err: %+v", err)
		}
		switch message.GetOperation() {
		case model.InsertOperation:
			err = dmiclient.DMIClientsImp.CreateDeviceModel(&dm)
			if err != nil {
				klog.Errorf("add device model %s failed with err: %v", dm.Name, err)
				return err
			}
			dw.dmiCache.DeviceModelMu.Lock()
			dw.dmiCache.DeviceModelList[dm.Name] = &dm
			dw.dmiCache.DeviceModelMu.Unlock()
		case model.DeleteOperation:
			err = dmiclient.DMIClientsImp.RemoveDeviceModel(&dm)
			if err != nil {
				klog.Errorf("delete device model %s failed with err: %v", dm.Name, err)
				return err
			}
			dw.dmiCache.DeviceModelMu.Lock()
			delete(dw.dmiCache.DeviceModelList, dm.Name)
			dw.dmiCache.DeviceModelMu.Unlock()
		case model.UpdateOperation:
			err = dmiclient.DMIClientsImp.UpdateDeviceModel(&dm)
			if err != nil {
				klog.Errorf("update device model %s failed with err: %v", dm.Name, err)
				return err
			}
			dw.dmiCache.DeviceModelMu.Lock()
			dw.dmiCache.DeviceModelList[dm.Name] = &dm
			dw.dmiCache.DeviceModelMu.Unlock()
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}

	default:
		klog.Warningf("unsupported resource type %s", resources[3])
	}

	return nil
}

func (dw *DMIWorker) initDeviceModelInfoFromDB() {
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDeviceModel)
	if err != nil {
		klog.Errorf("fail to init device model info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		deviceModel := v1alpha2.DeviceModel{}
		if err := json.Unmarshal([]byte(meta), &deviceModel); err != nil {
			klog.Errorf("fail to unmarshal device model info from db with err: %v", err)
			return
		}
		dw.dmiCache.DeviceModelMu.Lock()
		dw.dmiCache.DeviceModelList[deviceModel.Name] = &deviceModel
		dw.dmiCache.DeviceModelMu.Unlock()
	}
	klog.Infoln("success to init device model info from db")
}

func (dw *DMIWorker) initDeviceInfoFromDB() {
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDevice)
	if err != nil {
		klog.Errorf("fail to init device info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		device := v1alpha2.Device{}
		if err := json.Unmarshal([]byte(meta), &device); err != nil {
			klog.Errorf("fail to unmarshal device info from db with err: %v", err)
			return
		}
		dw.dmiCache.DeviceMu.Lock()
		dw.dmiCache.DeviceList[device.Name] = &device
		dw.dmiCache.DeviceMu.Unlock()
	}
	klog.Infoln("success to init device info from db")
}

func (dw *DMIWorker) initDeviceMapperInfoFromDB() {
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDeviceMapper)
	if err != nil {
		klog.Errorf("fail to init device mapper info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		deviceMapper := pb.MapperInfo{}
		if err := json.Unmarshal([]byte(meta), &deviceMapper); err != nil {
			klog.Errorf("fail to unmarshal device mapper info from db with err: %v", err)
			return
		}
		dw.dmiCache.MapperMu.Lock()
		dw.dmiCache.MapperList[deviceMapper.Name] = &deviceMapper
		dw.dmiCache.MapperMu.Unlock()
	}
	klog.Infoln("success to init device mapper info from db")
}
