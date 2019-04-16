package controller

import (
	"reflect"
	"strconv"
	"time"

	/*"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/controller/constants"*/
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/types"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/utils"

	"github.com/satori/go.uuid"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/apis/core"
)

// CacheDevice is the struct save device data for check device is really changed
type CacheDevice struct {
	metav1.ObjectMeta
	Spec   v1alpha1.DeviceSpec
	Status v1alpha1.DeviceStatus
}

// CacheDeviceModel is the struct save DeviceModel data for check DeviceModel is really changed
type CacheDeviceModel struct {
	metav1.ObjectMeta
	Spec v1alpha1.DeviceModelSpec
}

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient   *kubernetes.Clientset
	messageLayer messagelayer.MessageLayer

	deviceManager *manager.DeviceManager
	deviceStop    chan struct{}

	deviceModelManager *manager.DeviceModelManager
	deviceModelStop    chan struct{}

	crdClient *rest.RESTClient
	crdScheme *runtime.Scheme
}

func (dc *DownstreamController) syncDeviceModel(stop chan struct{}) {
	running := true
	for running {
		select {
		case e := <-dc.deviceModelManager.Events():
			deviceModel, ok := e.Object.(*v1alpha1.DeviceModel)
			if !ok {
				log.LOGGER.Warnf("object type: %T unsupported", deviceModel)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.deviceModelAdded(deviceModel)
			case watch.Deleted:
				dc.deviceModelDeleted(deviceModel)
			case watch.Modified:
				dc.deviceModelUpdated(deviceModel)
			default:
				log.LOGGER.Warnf("deviceModel event type: %s unsupported", e.Type)
			}
		case <-stop:
			log.LOGGER.Infof("stop syncDeviceModel")
			running = false
		}
	}
}

func (dc *DownstreamController) deviceModelAdded(deviceModel *v1alpha1.DeviceModel) {
	// TODO: looks like nothing to do when deviceModel added, only add in map
	dc.deviceModelManager.DeviceModel.Store(deviceModel.Name, &CacheDeviceModel{ObjectMeta: deviceModel.ObjectMeta, Spec: deviceModel.Spec})
}

func isDeviceModelUpdated(old *CacheDeviceModel, new *v1alpha1.DeviceModel) bool {
	// does not care fields
	old.ObjectMeta.ResourceVersion = new.ObjectMeta.ResourceVersion
	old.ObjectMeta.Generation = new.ObjectMeta.Generation

	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) || !reflect.DeepEqual(old.Spec, new.Spec)
}

func (dc *DownstreamController) deviceModelUpdated(deviceModel *v1alpha1.DeviceModel) {
	value, ok := dc.deviceModelManager.DeviceModel.Load(deviceModel.Name)
	dc.deviceModelManager.DeviceModel.Store(deviceModel.Name, &CacheDeviceModel{ObjectMeta: deviceModel.ObjectMeta, Spec: deviceModel.Spec})
	if ok {
		cachedDeviceModel := value.(*CacheDeviceModel)
		if isDeviceModelUpdated(cachedDeviceModel, deviceModel) {
			//TODO: add logic to update config map
		}
	} else {
		dc.deviceModelAdded(deviceModel)
	}
}

func (dc *DownstreamController) deviceModelDeleted(deviceModel *v1alpha1.DeviceModel) {
	// TODO: If finalizers used, all devices will be deleted before delete of deviceModel, then just delete from deviceModel map and config-map
}

func (dc *DownstreamController) syncDevice(stop chan struct{}) {
	running := true
	for running {
		select {
		case e := <-dc.deviceManager.Events():
			device, ok := e.Object.(*v1alpha1.Device)
			if !ok {
				log.LOGGER.Warnf("object type: %T unsupported", device)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.deviceAdded(device)
			case watch.Deleted:
				dc.deviceDeleted(device)
			case watch.Modified:
				dc.deviceUpdated(device)
			default:
				log.LOGGER.Warnf("device event type: %s unsupported", e.Type)
			}
		case <-stop:
			log.LOGGER.Infof("stop syncDevice")
			running = false
		}
	}
}

// deviceAdded creates a device, adds in deviceManagers map, send a message to edge node if node selector is present.
func (dc *DownstreamController) deviceAdded(device *v1alpha1.Device) {
	dc.deviceManager.Device.Store(device.Name, &CacheDevice{ObjectMeta: device.ObjectMeta, Spec: device.Spec, Status: device.Status})
	edgeDevice := createDevice(device)
	msg := model.NewMessage("")

	// TODO: Use node selector instead of hardcoding values, check if node is available
	resource, err := messagelayer.BuildResource("fb4ebb70-2783-42b8-b3ef-63e2fd6d242e", "membership", "")
	if err != nil {
		log.LOGGER.Warnf("built message resource failed with error: %s", err)
		return
	}
	msg.BuildRouter(constants.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)

	content := types.MembershipUpdate{AddDevices: []types.Device{
		edgeDevice,
	}}
	content.EventID = uuid.NewV4().String()
	content.Timestamp = time.Now().UnixNano() / 1e6
	msg.Content = content

	err = dc.messageLayer.Send(*msg)
	if err != nil {
		log.LOGGER.Errorf("Failed to send device addition message %v due to error %v", msg, err)
	}
}

func createDevice(device *v1alpha1.Device) types.Device {
	edgeDevice := types.Device{
		ID:   string(device.UID),
		Name: device.Name,
	}

	description, ok := device.Labels["description"]
	if ok {
		edgeDevice.Description = description
	}

	// TODO: optional is Always false, currently not present in CRD definition, need to add or remove from deviceTwin @ Edge
	opt := false
	optional := &opt
	twin := make(map[string]*types.MsgTwin)
	for _, dtwin := range device.Status.Twins {
		expected := &types.TwinValue{}
		expected.Value = &dtwin.Desired.Value
		timestamp := time.Now().UnixNano() / 1e6

		metadata := &types.ValueMetadata{Timestamp: timestamp}
		expected.Metadata = metadata

		// TODO: how to manage versioning ??
		cloudVersion, _ := strconv.ParseInt(device.ResourceVersion, 10, 64)
		twinVersion := &types.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
		msgTwin := &types.MsgTwin{
			Expected:        expected,
			Optional:        optional,
			Metadata:        &types.TypeMetadata{Type: "string"},
			ExpectedVersion: twinVersion,
		}
		twin[dtwin.PropertyName] = msgTwin
	}
	edgeDevice.Twin = twin
	return edgeDevice
}

func isDeviceUpdated(old *CacheDevice, new *v1alpha1.Device) bool {
	// does not care fields
	old.ObjectMeta.ResourceVersion = new.ObjectMeta.ResourceVersion
	old.ObjectMeta.Generation = new.ObjectMeta.Generation

	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) || !reflect.DeepEqual(old.Spec, new.Spec) || !reflect.DeepEqual(old.Status, new.Status)
}

func isNodeSelectorUpdated(old *core.NodeSelector, new *core.NodeSelector) bool {
	return !reflect.DeepEqual(old.NodeSelectorTerms, new.NodeSelectorTerms)
}

// deviceUpdated updates the map, check if device is actually updated.
// If nodeSelector is updated, call add device for newNode, deleteDevice for old Node.
// If twin is updated, send twin update message to edge
func (dc *DownstreamController) deviceUpdated(device *v1alpha1.Device) {
	value, ok := dc.deviceManager.Device.Load(device.Name)
	dc.deviceManager.Device.Store(device.Name, &CacheDevice{ObjectMeta: device.ObjectMeta, Spec: device.Spec, Status: device.Status})
	if ok {
		cachedDevice := value.(*CacheDevice)
		if isDeviceUpdated(cachedDevice, device) {
			// if node selector updated delete from old node and create in new node
			if isNodeSelectorUpdated(cachedDevice.Spec.NodeSelector, device.Spec.NodeSelector) {
				dc.deviceAdded(device)
				// TODO: add code to add in node's configmap
				deletedDevice := &v1alpha1.Device{ObjectMeta: cachedDevice.ObjectMeta,
					Spec:     cachedDevice.Spec,
					Status:   cachedDevice.Status,
					TypeMeta: device.TypeMeta,
				}
				dc.deviceDeleted(deletedDevice)
				// TODO: add code to delete from nodes configmap
			} else {
				// TODO: add logic to update the twins configMap
				twin := make(map[string]*types.MsgTwin)
				addUpdatedTwins(device.Status.Twins, twin, device.ResourceVersion)
				addDeletedTwins(cachedDevice.Status.Twins, device.Status.Twins, twin, device.ResourceVersion)
				msg := model.NewMessage("")

				// TODO: Use node selector instead of hardcoding values, check if node is available
				resource, err := messagelayer.BuildResource("fb4ebb70-2783-42b8-b3ef-63e2fd6d242e", "device/"+string(device.UID)+"/twin/cloud_updated", "")
				if err != nil {
					log.LOGGER.Warnf("built message resource failed with error: %s", err)
					return
				}
				msg.BuildRouter(constants.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)
				content := types.DeviceTwinUpdate{Twin: twin}
				content.EventID = uuid.NewV4().String()
				content.Timestamp = time.Now().UnixNano() / 1e6
				msg.Content = content

				err = dc.messageLayer.Send(*msg)
				if err != nil {
					log.LOGGER.Errorf("Failed to send deviceTwin message %v due to error %v", msg, err)
				}
			}
		}

	} else {
		// If device not present in device map means it is not modified and added.
		dc.deviceAdded(device)
	}
}

func addDeletedTwins(old []v1alpha1.Twin, new []v1alpha1.Twin, twin map[string]*types.MsgTwin, version string) {
	opt := false
	optional := &opt
	for _, dtwin := range old {
		if !ifTwinPresent(dtwin, new) {
			expected := &types.TwinValue{}
			expected.Value = &dtwin.Desired.Value
			timestamp := time.Now().UnixNano() / 1e6

			metadata := &types.ValueMetadata{Timestamp: timestamp}
			expected.Metadata = metadata

			// TODO: how to manage versioning ??
			cloudVersion, _ := strconv.ParseInt(version, 10, 64)
			twinVersion := &types.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
			msgTwin := &types.MsgTwin{
				Expected:        expected,
				Optional:        optional,
				Metadata:        &types.TypeMetadata{Type: "deleted"},
				ExpectedVersion: twinVersion,
			}
			twin[dtwin.PropertyName] = msgTwin
		}
	}
}

func ifTwinPresent(twin v1alpha1.Twin, new []v1alpha1.Twin) bool {
	for _, dtwin := range new {
		if twin.PropertyName == dtwin.PropertyName {
			return true
		}
	}
	return false
}

func addUpdatedTwins(new []v1alpha1.Twin, twin map[string]*types.MsgTwin, version string) {
	opt := false
	optional := &opt
	for _, dtwin := range new {
		expected := &types.TwinValue{}
		expected.Value = &dtwin.Desired.Value
		timestamp := time.Now().UnixNano() / 1e6

		metadata := &types.ValueMetadata{Timestamp: timestamp}
		expected.Metadata = metadata

		// TODO: how to manage versioning ??
		cloudVersion, _ := strconv.ParseInt(version, 10, 64)
		twinVersion := &types.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
		msgTwin := &types.MsgTwin{
			Expected:        expected,
			Optional:        optional,
			Metadata:        &types.TypeMetadata{Type: "string"},
			ExpectedVersion: twinVersion,
		}
		twin[dtwin.PropertyName] = msgTwin
	}
}

// deviceDeleted send a deleted message to the edgeNode and deletes the device from the deviceManager.Device map
func (dc *DownstreamController) deviceDeleted(device *v1alpha1.Device) {
	dc.deviceManager.Device.Delete(device.Name)
	edgeDevice := createDevice(device)
	msg := model.NewMessage("")

	// TODO: Use node selector instead of hardcoding values
	resource, err := messagelayer.BuildResource("fb4ebb70-2783-42b8-b3ef-63e2fd6d242e", "membership", "")
	msg.BuildRouter(constants.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)

	content := types.MembershipUpdate{RemoveDevices: []types.Device{
		edgeDevice,
	}}
	content.EventID = uuid.NewV4().String()
	content.Timestamp = time.Now().UnixNano() / 1e6
	msg.Content = content
	if err != nil {
		log.LOGGER.Warnf("built message resource failed with error: %s", err)
		return
	}
	err = dc.messageLayer.Send(*msg)
	if err != nil {
		log.LOGGER.Errorf("Failed to send device addition message %v due to error %v", msg, err)
	}
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	log.LOGGER.Infof("start downstream controller")

	dc.deviceModelStop = make(chan struct{})
	go dc.syncDeviceModel(dc.deviceModelStop)

	dc.deviceStop = make(chan struct{})
	go dc.syncDevice(dc.deviceStop)

	time.Sleep(100 * time.Second)
	return nil
}

// Stop DownstreamController
func (dc *DownstreamController) Stop() error {
	log.LOGGER.Infof("stop downstream controller")
	dc.deviceStop <- struct{}{}
	dc.deviceModelStop <- struct{}{}
	return nil
}

// NewDownstreamController create a DownstreamController from config
func NewDownstreamController() (*DownstreamController, error) {
	/*lc := &manager.LocationCache{}*/

	cli, err := utils.KubeClient()
	if err != nil {
		log.LOGGER.Warnf("create kube client failed with error: %s", err)
		return nil, err
	}
	config, err := utils.KubeConfig()
	crdcli, _, err := utils.NewCRDClient(config)
	deviceManager, err := manager.NewDeviceManager(crdcli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("create device manager failed with error: %s", err)
		return nil, err
	}

	deviceModelManager, err := manager.NewDeviceModelManager(crdcli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("create device manager failed with error: %s", err)
		return nil, err
	}

	ml, err := messagelayer.NewMessageLayer()
	if err != nil {
		log.LOGGER.Warnf("create message layer failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{kubeClient: cli, deviceManager: deviceManager, deviceModelManager: deviceModelManager, messageLayer: ml}

	return dc, nil
}
