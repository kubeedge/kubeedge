package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/modbus/driver"
	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/grpcclient"
	"github.com/kubeedge/mapper-framework/pkg/util/parse"
)

type TwinData struct {
	DeviceName      string
	DeviceNamespace string
	Client          *driver.CustomizedClient
	Name            string
	Type            string
	ObservedDesired common.TwinProperty
	VisitorConfig   *driver.VisitorConfig
	Topic           string
	Results         interface{}
	CollectCycle    time.Duration
	ReportToCloud   bool
}

func (td *TwinData) GetPayLoad() ([]byte, error) {
	var err error
	td.VisitorConfig.VisitorConfigData.DataType = strings.ToLower(td.VisitorConfig.VisitorConfigData.DataType)
	td.Results, err = td.Client.GetDeviceData(td.VisitorConfig)
	if err != nil {
		return nil, fmt.Errorf("get device data failed: %v", err)
	}
	sData, err := common.ConvertToString(td.Results)
	if err != nil {
		klog.Errorf("Failed to convert %s %s value as string : %v", td.DeviceName, td.Name, err)
		return nil, err
	}
	if len(sData) > 30 {
		klog.V(4).Infof("Get %s : %s ,value is %s......", td.DeviceName, td.Name, sData[:30])
	} else {
		klog.V(4).Infof("Get %s : %s ,value is %s", td.DeviceName, td.Name, sData)
	}
	var payload []byte
	if strings.Contains(td.Topic, "$hw") {
		if payload, err = common.CreateMessageTwinUpdate(td.Name, td.Type, sData, td.ObservedDesired.Value); err != nil {
			return nil, fmt.Errorf("create message twin update failed: %v", err)
		}
	} else {
		if payload, err = common.CreateMessageData(td.Name, td.Type, sData); err != nil {
			return nil, fmt.Errorf("create message data failed: %v", err)
		}
	}
	return payload, nil
}

func (td *TwinData) PushToEdgeCore() {
	payload, err := td.GetPayLoad()
	if err != nil {
		klog.Errorf("twindata %s unmarshal failed, err: %s", td.Name, err)
		return
	}

	var msg common.DeviceTwinUpdate
	if err = json.Unmarshal(payload, &msg); err != nil {
		klog.Errorf("twindata %s unmarshal failed, err: %s", td.Name, err)
		return
	}

	twins := parse.ConvMsgTwinToGrpc(msg.Twin)

	var rdsr = &dmiapi.ReportDeviceStatusRequest{
		DeviceName:      td.DeviceName,
		DeviceNamespace: td.DeviceNamespace,
		ReportedDevice: &dmiapi.DeviceStatus{
			Twins: twins,
			//State: "OK",
		},
	}

	if err := grpcclient.ReportDeviceStatus(rdsr); err != nil {
		klog.Errorf("fail to report device status of %s with err: %+v", rdsr.DeviceName, err)
	}
}

func (td *TwinData) Run(ctx context.Context) {
	if !td.ReportToCloud {
		return
	}
	if td.CollectCycle == 0 {
		td.CollectCycle = common.DefaultCollectCycle
	}
	ticker := time.NewTicker(td.CollectCycle)
	for {
		select {
		case <-ticker.C:
			td.PushToEdgeCore()
		case <-ctx.Done():
			return
		}
	}
}
