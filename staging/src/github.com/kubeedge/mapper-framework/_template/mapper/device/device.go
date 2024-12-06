package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	dbInflux "github.com/kubeedge/Template/data/dbmethod/influxdb2"
	dbMysql "github.com/kubeedge/Template/data/dbmethod/mysql"
	dbRedis "github.com/kubeedge/Template/data/dbmethod/redis"
	dbTdengine "github.com/kubeedge/Template/data/dbmethod/tdengine"
	httpMethod "github.com/kubeedge/Template/data/publish/http"
	mqttMethod "github.com/kubeedge/Template/data/publish/mqtt"
	otelMethod "github.com/kubeedge/Template/data/publish/otel"
	"github.com/kubeedge/Template/data/stream"
	"github.com/kubeedge/Template/driver"
	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/global"
	"github.com/kubeedge/mapper-framework/pkg/util/parse"
)

type DevPanel struct {
	deviceMuxs   map[string]context.CancelFunc
	devices      map[string]*driver.CustomizedDev
	models       map[string]common.DeviceModel
	wg           sync.WaitGroup
	serviceMutex sync.Mutex
	quitChan     chan os.Signal
}

var (
	devPanel *DevPanel
	once     sync.Once
)

var ErrEmptyData = errors.New("device or device model list is empty")

// NewDevPanel init and return devPanel
func NewDevPanel() *DevPanel {
	once.Do(func() {
		devPanel = &DevPanel{
			deviceMuxs:   make(map[string]context.CancelFunc),
			devices:      make(map[string]*driver.CustomizedDev),
			models:       make(map[string]common.DeviceModel),
			wg:           sync.WaitGroup{},
			serviceMutex: sync.Mutex{},
			quitChan:     make(chan os.Signal),
		}
	})
	return devPanel
}

// DevStart start all devices.
func (d *DevPanel) DevStart() {
	for id, dev := range d.devices {
		klog.V(4).Info("Dev: ", id, dev)
		ctx, cancel := context.WithCancel(context.Background())
		d.deviceMuxs[id] = cancel
		d.wg.Add(1)
		go d.start(ctx, dev)
	}
	signal.Notify(d.quitChan, os.Interrupt)
	go func() {
		<-d.quitChan
		for id, device := range d.devices {
			err := device.CustomizedClient.StopDevice()
			if err != nil {
				klog.Errorf("Service has stopped but failed to stop %s:%v", id, err)
			}
		}
		klog.V(1).Info("Exit mapper")
		os.Exit(1)
	}()
	d.wg.Wait()
}

// start the device
func (d *DevPanel) start(ctx context.Context, dev *driver.CustomizedDev) {
	defer d.wg.Done()

	var protocolConfig driver.ProtocolConfig
	if err := json.Unmarshal(dev.Instance.PProtocol.ConfigData, &protocolConfig); err != nil {
		klog.Errorf("Unmarshal ProtocolConfigs error: %v", err)
		return
	}
	client, err := driver.NewClient(protocolConfig)
	if err != nil {
		klog.Errorf("Init dev %s error: %v", dev.Instance.Name, err)
		return
	}
	dev.CustomizedClient = client
	err = dev.CustomizedClient.InitDevice()
	if err != nil {
		klog.Errorf("Init device %s error: %v", dev.Instance.ID, err)
		return
	}
	go dataHandler(ctx, dev)
	<-ctx.Done()
}

// dataHandler initialize the timer to handle data plane and devicetwin.
func dataHandler(ctx context.Context, dev *driver.CustomizedDev) {
	// handle device status report
	getStates := &DeviceStates{
		Client:          dev.CustomizedClient,
		DeviceName:      dev.Instance.Name,
		DeviceNamespace: dev.Instance.Namespace,
		ReportToCloud:   dev.Instance.Status.ReportToCloud,
		ReportCycle:     time.Millisecond * time.Duration(dev.Instance.Status.ReportCycle),
	}
	go getStates.Run(ctx)
	// handle device twin report
	for _, twin := range dev.Instance.Twins {
		twin.Property.PProperty.DataType = strings.ToLower(twin.Property.PProperty.DataType)
		var visitorConfig driver.VisitorConfig

		err := json.Unmarshal(twin.Property.Visitors, &visitorConfig)
		visitorConfig.VisitorConfigData.DataType = strings.ToLower(visitorConfig.VisitorConfigData.DataType)
		if err != nil {
			klog.Errorf("Unmarshal VisitorConfig error: %v", err)
			continue
		}
		err = setVisitor(&visitorConfig, &twin, dev)
		if err != nil {
			klog.Error(err)
			continue
		}

		// If the device property type is streaming, it will directly enter the streaming data processing function,
		// such as saving frames or saving videos, and will no longer push it to the user database and application.
		// If there are other needs for stream data processing, users can add functions in the mapper/data/stream directory.
		if twin.Property.PProperty.DataType == "stream" {
			err = stream.StreamHandler(&twin, dev.CustomizedClient, &visitorConfig)
			if err != nil {
				klog.Errorf("processed streaming data by %s Error: %v", twin.PropertyName, err)
			}
			continue
		}

		// handle twin
		twinData := &TwinData{
			DeviceName:      dev.Instance.Name,
			DeviceNamespace: dev.Instance.Namespace,
			Client:          dev.CustomizedClient,
			Name:            twin.PropertyName,
			Type:            twin.ObservedDesired.Metadata.Type,
			ObservedDesired: twin.ObservedDesired,
			VisitorConfig:   &visitorConfig,
			Topic:           fmt.Sprintf(common.TopicTwinUpdate, dev.Instance.ID),
			CollectCycle:    time.Millisecond * time.Duration(twin.Property.CollectCycle),
			ReportToCloud:   twin.Property.ReportToCloud,
		}
		go twinData.Run(ctx)

		dataModel := common.NewDataModel(dev.Instance.Name, twin.Property.PropertyName, dev.Instance.Namespace, common.WithType(twin.ObservedDesired.Metadata.Type))
		// handle push method
		if twin.Property.PushMethod.MethodConfig != nil && twin.Property.PushMethod.MethodName != "" {
			pushHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
		}
		// handle database
		if twin.Property.PushMethod.DBMethod.DBMethodName != "" {
			dbHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
			switch twin.Property.PushMethod.DBMethod.DBMethodName {
			// TODO add more database
			case "influx":
				dbInflux.DataHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
			case "redis":
				dbRedis.DataHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
			case "tdengine":
				dbTdengine.DataHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
			case "mysql":
				dbMysql.DataHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
			}
		}
	}
}

// pushHandler start data panel work
func pushHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	if twin.Property.PushMethod.MethodName == common.PushMethodOTEL {
		otelMethod.DataHandler(ctx, twin, client, visitorConfig, dataModel)
		return
	}

	var dataPanel global.DataPanel
	var err error
	// initialization dataPanel
	switch twin.Property.PushMethod.MethodName {
	case common.PushMethodHTTP:
		dataPanel, err = httpMethod.NewDataPanel(twin.Property.PushMethod.MethodConfig)
	case common.PushMethodMQTT:
		dataPanel, err = mqttMethod.NewDataPanel(twin.Property.PushMethod.MethodConfig)
	default:
		err = errors.New("custom protocols are not currently supported when push data")
	}
	if err != nil {
		klog.Errorf("new data panel error: %v", err)
		return
	}
	// initialization PushMethod
	err = dataPanel.InitPushMethod()
	if err != nil {
		klog.Errorf("init publish method err: %v", err)
		return
	}
	reportCycle := time.Millisecond * time.Duration(twin.Property.ReportCycle)
	if reportCycle == 0 {
		reportCycle = common.DefaultReportCycle
	}
	ticker := time.NewTicker(reportCycle)
	go func() {
		for {
			select {
			case <-ticker.C:
				deviceData, err := client.GetDeviceData(visitorConfig)
				if err != nil {
					klog.Errorf("publish error: %v", err)
					continue
				}
				sData, err := common.ConvertToString(deviceData)
				if err != nil {
					klog.Errorf("Failed to convert publish method data : %v", err)
					continue
				}
				dataModel.SetValue(sData)
				dataModel.SetTimeStamp()
				dataPanel.Push(dataModel)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// dbHandler start db client to save data
func dbHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	switch twin.Property.PushMethod.DBMethod.DBMethodName {
	// TODO add more database
	case "influx":
		dbInflux.DataHandler(ctx, twin, client, visitorConfig, dataModel)

	case "redis":
		dbRedis.DataHandler(ctx, twin, client, visitorConfig, dataModel)

	case "tdengine":
		dbTdengine.DataHandler(ctx, twin, client, visitorConfig, dataModel)

	case "mysql":
		dbMysql.DataHandler(ctx, twin, client, visitorConfig, dataModel)
	}
}

// setVisitor check if visitor property is readonly, if not then set it.
func setVisitor(visitorConfig *driver.VisitorConfig, twin *common.Twin, dev *driver.CustomizedDev) error {
	if twin.Property.PProperty.AccessMode == "ReadOnly" {
		klog.V(3).Infof("%s twin readonly property: %s", dev.Instance.Name, twin.PropertyName)
		return nil
	}
	klog.V(2).Infof("Convert type: %s, value: %s ", twin.Property.PProperty.DataType, twin.ObservedDesired.Value)
	var value interface{}
	if twin.ObservedDesired.Value != "" {
		convertedValue, err := common.Convert(twin.Property.PProperty.DataType, twin.ObservedDesired.Value)
		if err != nil {
			klog.Errorf("Failed to convert value as %s : %v", twin.Property.PProperty.DataType, err)
			return err
		}
		value = convertedValue
	} else {
		value = twin.ObservedDesired.Value
	}
	err := dev.CustomizedClient.SetDeviceData(value, visitorConfig)
	if err != nil {
		return fmt.Errorf("%s set device data error: %v", twin.PropertyName, err)
	}
	return nil
}

// DevInit initialize the device
func (d *DevPanel) DevInit(deviceList []*dmiapi.Device, deviceModelList []*dmiapi.DeviceModel) error {
	if len(deviceList) == 0 || len(deviceModelList) == 0 {
		return ErrEmptyData
	}

	for i := range deviceModelList {
		model := deviceModelList[i]
		cur := parse.GetDeviceModelFromGrpc(model)
		d.models[model.Name] = cur
	}

	for i := range deviceList {
		device := deviceList[i]
		commonModel := d.models[device.Spec.DeviceModelReference]
		protocol, err := parse.BuildProtocolFromGrpc(device)
		if err != nil {
			return err
		}
		instance, err := parse.GetDeviceFromGrpc(device, &commonModel)
		if err != nil {
			return err
		}
		instance.PProtocol = protocol

		cur := new(driver.CustomizedDev)
		cur.Instance = *instance
		d.devices[instance.ID] = cur
	}

	return nil
}

// UpdateDev stop old device, then update and start new device
func (d *DevPanel) UpdateDev(model *common.DeviceModel, device *common.DeviceInstance) {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()

	if oldDevice, ok := d.devices[device.ID]; ok {
		err := d.stopDev(oldDevice, device.ID)
		if err != nil {
			klog.Error(err)
		}
	}
	// start new device
	d.devices[device.ID] = new(driver.CustomizedDev)
	d.devices[device.ID].Instance = *device
	d.models[model.ID] = *model

	ctx, cancelFunc := context.WithCancel(context.Background())
	d.deviceMuxs[device.ID] = cancelFunc
	d.wg.Add(1)
	go d.start(ctx, d.devices[device.ID])
}

// UpdateDevTwins update device's twins
func (d *DevPanel) UpdateDevTwins(deviceID string, twins []common.Twin) error {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	dev, ok := d.devices[deviceID]
	if !ok {
		return fmt.Errorf("device %s not found", deviceID)
	}
	dev.Instance.Twins = twins
	model := d.models[dev.Instance.Model]
	d.UpdateDev(&model, &dev.Instance)

	return nil
}

// DealDeviceTwinGet get device's twin data
func (d *DevPanel) DealDeviceTwinGet(deviceID string, twinName string) (interface{}, error) {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	dev, ok := d.devices[deviceID]
	if !ok {
		return nil, fmt.Errorf("not found device %s", deviceID)
	}
	var res []parse.TwinResultResponse
	for _, twin := range dev.Instance.Twins {
		if twinName != "" && twin.PropertyName != twinName {
			continue
		}
		payload, err := getTwinData(deviceID, twin, d.devices[deviceID])
		if err != nil {
			return nil, err
		}
		item := parse.TwinResultResponse{
			PropertyName: twinName,
			Payload:      payload,
		}
		res = append(res, item)
	}
	return json.Marshal(res)
}

// getTwinData get twin
func getTwinData(deviceID string, twin common.Twin, dev *driver.CustomizedDev) ([]byte, error) {
	var visitorConfig driver.VisitorConfig
	err := json.Unmarshal(twin.Property.Visitors, &visitorConfig)
	if err != nil {
		return nil, err
	}
	err = setVisitor(&visitorConfig, &twin, dev)
	if err != nil {
		return nil, err
	}
	twinData := &TwinData{
		DeviceName:    deviceID,
		Client:        dev.CustomizedClient,
		Name:          twin.PropertyName,
		Type:          twin.ObservedDesired.Metadata.Type,
		VisitorConfig: &visitorConfig,
		Topic:         fmt.Sprintf(common.TopicTwinUpdate, deviceID),
	}
	return twinData.GetPayLoad()
}

// GetDevice get device instance
func (d *DevPanel) GetDevice(deviceID string) (interface{}, error) {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	found, ok := d.devices[deviceID]
	if !ok || found == nil {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	// get the latest reported twin value
	for i, twin := range found.Instance.Twins {
		payload, err := getTwinData(deviceID, twin, found)
		if err != nil {
			return nil, err
		}
		found.Instance.Twins[i].Reported.Value = string(payload)
	}
	return found, nil
}

// RemoveDevice remove device instance
func (d *DevPanel) RemoveDevice(deviceID string) error {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	dev := d.devices[deviceID]
	delete(d.devices, deviceID)
	err := d.stopDev(dev, deviceID)
	if err != nil {
		return err
	}
	return nil
}

// WriteDevice write value to the device
func (d *DevPanel) WriteDevice(deviceMethodName, deviceID, propertyName, data string) error {
	var dataType string
	var deviceproperty common.DeviceProperty
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	dev, ok := d.devices[deviceID]
	if !ok {
		return fmt.Errorf("not found device %s", deviceID)
	}

	deviceMethodMap := make(map[string][]string)

	// get all deviceMethod of the device
	for _, method := range dev.Instance.Methods {
		deviceMethodMap[method.Name] = append(deviceMethodMap[method.Name], method.PropertyNames...)
	}
	// Determine whether the called device method exists
	propertyNames, ok := deviceMethodMap[deviceMethodName]
	if !ok {
		return fmt.Errorf("deviceMethod name %s does not exist in device instance", deviceMethodName)
	}
	// Determine whether the device property to be written is in the list defined by the device method
	flag := false
	for _, name := range propertyNames {
		if name == propertyName {
			flag = true
			break
		}
	}
	if !flag {
		return fmt.Errorf("deviceProperty %s to be written is not in the list defined by devicemethod", propertyName)
	}
	// Determine whether the device property to be written is in the device instance
	flag = false
	for _, property := range dev.Instance.Properties {
		if property.PropertyName != propertyName {
			continue
		}
		dataType = property.PProperty.DataType
		deviceproperty = property
		flag = true
		break
	}
	if !flag {
		return fmt.Errorf("can't find device propertyName %s in device instance", propertyName)
	}
	klog.V(2).Infof("start writing values %v to device %s property %s", data, deviceID, propertyName)
	writeData, err := common.Convert(strings.ToLower(dataType), data)
	if err != nil {
		return fmt.Errorf("conversion data format failed, datatype is %s, data is %s", strings.ToLower(dataType), data)
	}
	var visitorConfig driver.VisitorConfig
	err = json.Unmarshal(deviceproperty.Visitors, &visitorConfig)
	if err != nil {
		return err
	}

	err = dev.CustomizedClient.DeviceDataWrite(&visitorConfig, deviceMethodName, propertyName, writeData)
	if err != nil {
		return err
	}
	return nil
}

// stopDev stop device and goroutine
func (d *DevPanel) stopDev(dev *driver.CustomizedDev, id string) error {
	cancelFunc, ok := d.deviceMuxs[id]
	if !ok {
		return fmt.Errorf("can not find device %s from device muxs", id)
	}

	err := dev.CustomizedClient.StopDevice()
	if err != nil {
		klog.Errorf("stop device %s error: %v", id, err)
	}
	cancelFunc()
	return nil
}

// GetModel if the model exists, return device model
func (d *DevPanel) GetModel(modelID string) (common.DeviceModel, error) {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	if model, ok := d.models[modelID]; ok {
		return model, nil
	}
	return common.DeviceModel{}, fmt.Errorf("deviceModel %s not found", modelID)
}

// UpdateModel update device model
func (d *DevPanel) UpdateModel(model *common.DeviceModel) {
	d.serviceMutex.Lock()
	d.models[model.ID] = *model
	d.serviceMutex.Unlock()
}

// RemoveModel remove device model
func (d *DevPanel) RemoveModel(modelID string) {
	d.serviceMutex.Lock()
	delete(d.models, modelID)
	d.serviceMutex.Unlock()
}

// GetTwinResult Get twin's value and data type
func (d *DevPanel) GetTwinResult(deviceID string, twinName string) (string, string, error) {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	dev, ok := d.devices[deviceID]
	if !ok {
		return "", "", fmt.Errorf("not found device %s", deviceID)
	}
	var res string
	var dataType string
	for _, twin := range dev.Instance.Twins {
		if twinName != "" && twin.PropertyName != twinName {
			continue
		}
		var visitorConfig driver.VisitorConfig
		err := json.Unmarshal(twin.Property.Visitors, &visitorConfig)
		if err != nil {
			return "", "", err
		}
		err = setVisitor(&visitorConfig, &twin, dev)

		data, err := dev.CustomizedClient.GetDeviceData(&visitorConfig)
		if err != nil {
			return "", "", fmt.Errorf("get device data failed: %v", err)
		}
		res, err = common.ConvertToString(data)
		if err != nil {
			return "", "", err
		}
		dataType = twin.Property.PProperty.DataType
	}
	return res, dataType, nil
}

// GetDeviceMethod get method and property dataType of device
func (d *DevPanel) GetDeviceMethod(deviceID string) (map[string][]string, map[string]string, error) {
	klog.V(2).Infof("starting get method and property dataType of device %s", deviceID)
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	found, ok := d.devices[deviceID]
	if !ok || found == nil {
		return nil, nil, fmt.Errorf("device %s not found", deviceID)
	}

	deviceMethodMap := make(map[string][]string)
	propertyTypeMap := make(map[string]string)

	// get all deviceMethod of the device
	for _, method := range found.Instance.Methods {
		deviceMethodMap[method.Name] = append(deviceMethodMap[method.Name], method.PropertyNames...)
	}

	// get all deviceProperty type of the device
	for _, property := range found.Instance.Properties {
		propertyTypeMap[property.Name] = strings.ToLower(property.PProperty.DataType) // The original data type is an uppercase form such as INT FLOAT and needs to be converted.
	}
	return deviceMethodMap, propertyTypeMap, nil
}
