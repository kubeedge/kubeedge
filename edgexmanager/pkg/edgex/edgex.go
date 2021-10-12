package edgex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	clientsHttp "github.com/edgexfoundry/go-mod-core-contracts/v2/v2/clients/http"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/clients/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos/requests"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"

	devicesv1alpha3 "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha3"
)

type EdgeXClient struct {
	metadataUrl         string
	commandUrl          string
	deviceClient        interfaces.DeviceClient
	deviceProfileClient interfaces.DeviceProfileClient
	deviceCommandClient interfaces.DeviceServiceCommandClient
}

func NewEdgeXClient(metadataUrl, commandUrl string) *EdgeXClient {
	client := &EdgeXClient{
		metadataUrl: metadataUrl,
		commandUrl:  commandUrl,
	}
	client.deviceClient = clientsHttp.NewDeviceClient(metadataUrl)
	client.deviceProfileClient = clientsHttp.NewDeviceProfileClient(metadataUrl)
	client.deviceCommandClient = clientsHttp.NewDeviceServiceCommandClient()
	return client
}

func (edgex *EdgeXClient) GetDeviceByName(ctx context.Context, deviceName string) (string, error) {
	response, err := edgex.deviceClient.DeviceByName(ctx, deviceName)
	if err != nil {
		fmt.Printf("get device error: %v", err)
		if err.Code() == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}
	fmt.Printf("get device: %v", response)
	return response.Device.Id, nil
}

func (edgex *EdgeXClient) AddDevice(ctx context.Context, device *dtos.Device) (string, error) {
	req := requests.NewAddDeviceRequest(*device)
	response, err := edgex.deviceClient.Add(ctx, []requests.AddDeviceRequest{req})
	if err != nil {
		return "", err
	}
	if response[0].StatusCode != http.StatusCreated {
		return "", errors.New(response[0].Message)
	}
	return response[0].Id, nil
}

func (edgex *EdgeXClient) DeleteDeviceByName(ctx context.Context, deviceName string) error {
	if _, err := edgex.deviceClient.DeleteDeviceByName(ctx, deviceName); err != nil && err.Code() != http.StatusNotFound {
		return err
	}
	return nil
}

func (edgex *EdgeXClient) UpdateDevice(ctx context.Context, device *dtos.UpdateDevice) error {
	req := requests.NewUpdateDeviceRequest(*device)
	response, err := edgex.deviceClient.Update(ctx, []requests.UpdateDeviceRequest{req})
	if err != nil {
		return err
	} else if response[0].StatusCode != http.StatusOK {
		return errors.New(response[0].Message)
	}
	return nil
}

func (edgex *EdgeXClient) GetDeviceProfileByName(ctx context.Context, deviceProfileName string) (string, error) {
	response, err := edgex.deviceProfileClient.DeviceProfileByName(ctx, deviceProfileName)
	if err != nil {
		if err.Code() == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}
	return response.Profile.Id, nil

}

func (edgex *EdgeXClient) AddDeviceProfile(ctx context.Context, deviceProfile *dtos.DeviceProfile) (string, error) {
	req := requests.NewDeviceProfileRequest(*deviceProfile)
	response, err := edgex.deviceProfileClient.Add(ctx, []requests.DeviceProfileRequest{req})
	if err != nil {
		return "", err
	}
	if response[0].StatusCode != http.StatusCreated {
		return "", errors.New(response[0].Message)
	}
	return response[0].Id, nil
}

func (edgex *EdgeXClient) UpdateDeviceProfile(ctx context.Context, deviceProfile *dtos.DeviceProfile) error {
	req := requests.NewDeviceProfileRequest(*deviceProfile)
	response, err := edgex.deviceProfileClient.Update(ctx, []requests.DeviceProfileRequest{req})
	if err != nil {
		return err
	} else if response[0].StatusCode != http.StatusOK {
		return errors.New(response[0].Message)
	}
	return nil
}

func (edgex *EdgeXClient) DeleteDeviceProfileByName(ctx context.Context, deviceProfileName string) error {
	if _, err := edgex.deviceProfileClient.DeleteByName(ctx, deviceProfileName); err != nil && err.Code() != http.StatusNotFound {
		return err
	}
	return nil
}

func (edgex *EdgeXClient) GetDeviceResourceByName(ctx context.Context, deviceName, deviceResourceName string) (string, error) {
	res, err := edgex.deviceCommandClient.GetCommand(ctx, edgex.commandUrl, deviceName, deviceResourceName, "")
	if err != nil {
		return "", err
	}
	return res.Event.Readings[0].SimpleReading.Value, nil
}

func (edgex *EdgeXClient) SetDeviceResourceByName(ctx context.Context, deviceName, deviceResourceName string, body map[string]string) error {
	fmt.Println(body)
	_, err := edgex.deviceCommandClient.SetCommand(ctx, edgex.commandUrl, deviceName, deviceResourceName, "", body)
	if err != nil {
		return err
	}
	return nil
}

func ConvertToDeviceDTO(device *devicesv1alpha3.Device) (*dtos.Device, error) {
	var edgexDevice dtos.Device
	//edgexDevice.Id = device.Id
	edgexDevice.Name = device.Namespace + "-" + device.Name
	edgexDevice.Description = device.Name
	svc := device.Spec.DeviceService["deviceService"]
	var svcName string
	if err:=json.Unmarshal(svc.Raw, &svcName);err!=nil{
		return nil,err
	}
	edgexDevice.ServiceName = svcName
	edgexDevice.ProfileName = device.Spec.ModelRef
	edgexDevice.AdminState = "UNLOCKED"
	edgexDevice.OperatingState = "UP"
	//edgexDevice.LastReported = device.LastReported
	//edgexDevice.LastConnected = device.LastConnected
	//edgexDevice.Labels = device.Labels
	//edgexDevice.Location = device.Location
	//edgexDevice.AutoEvents = FromAutoEventModelsToDTOs(d.AutoEvents)
	if device.Spec.Protocol.Args != nil {
		p := dtos.ProtocolProperties{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(device.Spec.Protocol.Args.UnstructuredContent(), &p); err != nil {
			return nil, err
		}
		edgexDevice.Protocols = map[string]dtos.ProtocolProperties{
			device.Spec.Protocol.Name: p,
		}
	}
	edgexDevice.Protocols = map[string]dtos.ProtocolProperties{
		device.Spec.Protocol.Name: {
			"Address":  device.Spec.Protocol.Address,
			"Protocol": device.Spec.Protocol.Type,
		},
	}
	fmt.Printf("%+v \n", edgexDevice)
	return &edgexDevice, nil
}

func ConvertToUpdateDeviceDTO(device *devicesv1alpha3.Device) (*dtos.UpdateDevice, error) {
	s := func(i string) *string { return &i }
	var edgexDevice dtos.UpdateDevice
	//edgexDevice.Id = device.Id
	edgexDevice.Name = s(device.Namespace + "-" + device.Name)
	edgexDevice.Description = &device.Name
	svc := device.Spec.DeviceService["deviceService"]
	var svcName string
	if err:=json.Unmarshal(svc.Raw, &svcName);err!=nil{
		return nil,err
	}
	edgexDevice.ServiceName = s(svcName)
	edgexDevice.ProfileName = &device.Spec.ModelRef
	edgexDevice.AdminState = s("UNLOCKED")
	edgexDevice.OperatingState = s("UP")
	//edgexDevice.LastReported = device.LastReported
	//edgexDevice.LastConnected = device.LastConnected
	//edgexDevice.Labels = device.Labels
	//edgexDevice.Location = device.Location
	//edgexDevice.AutoEvents = FromAutoEventModelsToDTOs(d.AutoEvents)
	if device.Spec.Protocol.Args != nil {
		p := dtos.ProtocolProperties{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(device.Spec.Protocol.Args.UnstructuredContent(), &p); err != nil {
			return nil, err
		}
		edgexDevice.Protocols = map[string]dtos.ProtocolProperties{
			device.Spec.Protocol.Name: p,
		}
	}
	edgexDevice.Protocols = map[string]dtos.ProtocolProperties{
		device.Spec.Protocol.Name: {
			"Address":  device.Spec.Protocol.Address,
			"Protocol": device.Spec.Protocol.Type,
		},
	}
	fmt.Printf("%+v \n", edgexDevice)
	return &edgexDevice, nil
}

func ConvertToDeviceProfileDTO(deviceProfile *devicesv1alpha3.DeviceModel, visitors *devicesv1alpha3.DeviceAccess) (*dtos.DeviceProfile, error) {
	edgexProfile := &dtos.DeviceProfile{}
	//edgexProfile.Id = deviceProfile.Name
	edgexProfile.Name = deviceProfile.Name
	//edgexProfile.Description = deviceProfile.
	//edgexProfile.Manufacturer = deviceProfile.Manufacturer
	//edgexProfile.Model = deviceProfile.Model
	//edgexProfile.Labels = deviceProfile.Labels
	edgexProfile.DeviceResources = make([]dtos.DeviceResource, len(deviceProfile.Spec.DeviceProperties))
	for i := 0; i < len(edgexProfile.DeviceResources); i++ {
		r := ConvertToResProp(&deviceProfile.Spec.DeviceProperties[i], &visitors.Spec.AccessParameters[i])
		edgexProfile.DeviceResources[i] = *r
	}
	//edgexProfile.DeviceCommands = de
	fmt.Printf("%+v \n", edgexProfile)
	return edgexProfile, nil
}

func ConvertToResProp(r *devicesv1alpha3.DeviceProperties, v *devicesv1alpha3.AccessParameter) *dtos.DeviceResource {
	var resProp dtos.DeviceResource
	resProp.Name = r.Name
	resProp.Description = r.Description
	resProp.IsHidden = r.Mutable
	resProp.Properties.ReadWrite = r.ReadWrite
	resProp.Properties.DefaultValue = r.DefaultValue
	resProp.Properties.ValueType = string(r.Type)
	resProp.Properties.Maximum = r.Maximum
	resProp.Properties.Minimum = r.Minimum
	resProp.Attributes = make(map[string]interface{}, len(v.Parameter))
	for k, v := range v.Parameter {
		resProp.Attributes[k] = v
	}
	return &resProp
}
