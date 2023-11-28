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
	dbOpenGemini "github.com/kubeedge/Template/data/dbmethod/openGemini"
	httpMethod "github.com/kubeedge/Template/data/publish/http"
	mqttMethod "github.com/kubeedge/Template/data/publish/mqtt"
	"github.com/kubeedge/Template/driver"
	"github.com/kubeedge/Template/pkg/common"
	"github.com/kubeedge/Template/pkg/config"
	"github.com/kubeedge/Template/pkg/global"
	"github.com/kubeedge/Template/pkg/util/parse"
)

type DevPanel struct {
	deviceMuxs   map[string]context.CancelFunc
	devices      map[string]*driver.CustomizedDev
	models       map[string]common.DeviceModel
	protocols    map[string]common.ProtocolConfig
	wg           sync.WaitGroup
	serviceMutex sync.Mutex
	quitChan     chan os.Signal
}

var (
	devPanel *DevPanel
	once     sync.Once
)

// NewDevPanel init and return devPanel
func NewDevPanel() *DevPanel {
	once.Do(func() {
		devPanel = &DevPanel{
			deviceMuxs:   make(map[string]context.CancelFunc),
			devices:      make(map[string]*driver.CustomizedDev),
			models:       make(map[string]common.DeviceModel),
			protocols:    make(map[string]common.ProtocolConfig),
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
		// handle twin
		twinData := &TwinData{
			DeviceName:      dev.Instance.Name,
			Client:          dev.CustomizedClient,
			Name:            twin.PropertyName,
			Type:            twin.ObservedDesired.Metadata.Type,
			ObservedDesired: twin.ObservedDesired,
			VisitorConfig:   &visitorConfig,
			Topic:           fmt.Sprintf(common.TopicTwinUpdate, dev.Instance.ID),
			CollectCycle:    time.Duration(twin.Property.CollectCycle),
			ReportToCloud:   twin.Property.ReportToCloud,
		}
		go twinData.Run(ctx)
		// handle push method
		if twin.Property.PushMethod.MethodConfig != nil && twin.Property.PushMethod.MethodName != "" {
			dataModel := common.NewDataModel(dev.Instance.Name, twin.Property.PropertyName, common.WithType(twin.ObservedDesired.Metadata.Type))
			pushHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
		}
		// handle database
		if twin.Property.PushMethod.DBMethod.DBMethodName != "" {
			dataModel := common.NewDataModel(dev.Instance.Name, twin.Property.PropertyName, common.WithType(twin.ObservedDesired.Metadata.Type))
			dbHandler(ctx, &twin, dev.CustomizedClient, &visitorConfig, dataModel)
		}
	}
}

// pushHandler start data panel work
func pushHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	var dataPanel global.DataPanel
	var err error
	// initialization dataPanel
	switch twin.Property.PushMethod.MethodName {
	case "http":
		dataPanel, err = httpMethod.NewDataPanel(twin.Property.PushMethod.MethodConfig)
	case "mqtt":
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
	reportCycle := time.Duration(twin.Property.ReportCycle)
	if reportCycle == 0 {
		reportCycle = 1 * time.Second
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
		dbConfig, err := dbInflux.NewDataBaseClient(twin.Property.PushMethod.DBMethod.DBConfig.Influxdb2ClientConfig, twin.Property.PushMethod.DBMethod.DBConfig.Influxdb2DataConfig)
		if err != nil {
			klog.Errorf("new database client error: %v", err)
			return
		}
		dbClient := dbConfig.InitDbClient()
		if err != nil {
			klog.Errorf("init database client err: %v", err)
			return
		}
		reportCycle := time.Duration(twin.Property.ReportCycle)
		if reportCycle == 0 {
			reportCycle = 1 * time.Second
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

					err = dbConfig.AddData(dataModel, dbClient)
					if err != nil {
						klog.Errorf("influx database add data error: %v", err)
						return
					}
				case <-ctx.Done():
					dbConfig.CloseSession(dbClient)
					return
				}
			}
		}()
	case "openGemini":
		dbConfig, err := dbOpenGemini.NewDataBaseClient(twin.Property.PushMethod.DBMethod.DBConfig.OpenGeminiClientConfig, twin.Property.PushMethod.DBMethod.DBConfig.OpenGeminiDataConfig)
		if err != nil {
			klog.Errorf("new openGemini database client error: %v", err)
			return
		}
		dbClient, err := dbConfig.InitDbClient()
		if err != nil {
			klog.Errorf("init openGemini database client err: %v", err)
			return
		}
		reportCycle := time.Duration(twin.Property.ReportCycle)
		if reportCycle == 0 {
			reportCycle = 1 * time.Second
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

					err = dbConfig.AddData(dataModel, dbClient)
					if err != nil {
						klog.Errorf("openGemini database add data error: %v", err)
						return
					}
				case <-ctx.Done():
					dbConfig.CloseSession(dbClient)
					return
				}
			}
		}()
	}
}

// setVisitor check if visitor property is readonly, if not then set it.
func setVisitor(visitorConfig *driver.VisitorConfig, twin *common.Twin, dev *driver.CustomizedDev) error {
	if twin.Property.PProperty.AccessMode == "ReadOnly" {
		klog.V(3).Infof("%s twin readonly property: %s", dev.Instance.Name, twin.PropertyName)
		return nil
	}
	klog.V(2).Infof("Convert type: %s, value: %s ", twin.Property.PProperty.DataType, twin.ObservedDesired.Value)
	value, err := common.Convert(twin.Property.PProperty.DataType, twin.ObservedDesired.Value)
	if err != nil {
		klog.Errorf("Failed to convert value as %s : %v", twin.Property.PProperty.DataType, err)
		return err
	}
	err = dev.CustomizedClient.SetDeviceData(value, visitorConfig)
	if err != nil {
		return fmt.Errorf("%s set device data error: %v", twin.PropertyName, err)
	}
	return nil
}

// DevInit initialize the device
func (d *DevPanel) DevInit(cfg *config.Config) error {
	devs := make(map[string]*common.DeviceInstance)

	switch cfg.DevInit.Mode {
	//case common.DevInitModeConfigmap:
	//	if err := parse.Parse(cfg.DevInit.Configmap, devs, d.models, d.protocols); err != nil {
	//		return err
	//	}
	case common.DevInitModeRegister:
		if err := parse.ParseByUsingRegister(devs, d.models, d.protocols); err != nil {
			return err
		}
	}

	for key, deviceInstance := range devs {
		cur := new(driver.CustomizedDev)
		cur.Instance = *deviceInstance
		d.devices[key] = cur
	}
	return nil
}

// UpdateDev stop old device, then update and start new device
func (d *DevPanel) UpdateDev(model *common.DeviceModel, device *common.DeviceInstance, protocol *common.ProtocolConfig) {
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
	d.models[device.ID] = *model
	d.protocols[device.ID] = *protocol

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
	protocol := d.protocols[dev.Instance.ProtocolName]
	d.UpdateDev(&model, &dev.Instance, &protocol)
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
func (d *DevPanel) GetModel(modelName string) (common.DeviceModel, error) {
	d.serviceMutex.Lock()
	defer d.serviceMutex.Unlock()
	if model, ok := d.models[modelName]; ok {
		return model, nil
	}
	return common.DeviceModel{}, fmt.Errorf("deviceModel %s not found", modelName)
}

// UpdateModel update device model
func (d *DevPanel) UpdateModel(model *common.DeviceModel) {
	d.serviceMutex.Lock()
	d.models[model.Name] = *model
	d.serviceMutex.Unlock()
}

// RemoveModel remove device model
func (d *DevPanel) RemoveModel(modelName string) {
	d.serviceMutex.Lock()
	delete(d.models, modelName)
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
