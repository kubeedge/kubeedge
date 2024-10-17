package httpserver

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/util/parse"
)

func (rs *RestServer) Ping(writer http.ResponseWriter, request *http.Request) {
	response := &PingResponse{
		BaseResponse: NewBaseResponse(http.StatusOK),
		Message:      fmt.Sprintf("This is %s API, the server is running normally.", APIVersion),
	}
	rs.sendResponse(writer, request, response, http.StatusOK)
}

func (rs *RestServer) DeviceRead(writer http.ResponseWriter, request *http.Request) {
	urlItem := strings.Split(request.URL.Path, "/")
	deviceNamespace := urlItem[len(urlItem)-3]
	deviceName := urlItem[len(urlItem)-2]
	propertyName := urlItem[len(urlItem)-1]
	deviceID := parse.GetResourceID(deviceNamespace, deviceName)
	res, dataType, err := rs.devPanel.GetTwinResult(deviceID, propertyName)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Get device data error: %v", err), http.StatusInternalServerError)
	} else {
		response := &DeviceReadResponse{
			BaseResponse: NewBaseResponse(http.StatusOK),
			Data: common.NewDataModel(
				deviceName,
				propertyName,
				deviceNamespace,
				common.WithValue(res),
				common.WithType(dataType),
			),
		}
		rs.sendResponse(writer, request, response, http.StatusOK)
	}
}

// GetDeviceMethod get all methods of the specified device
func (rs *RestServer) GetDeviceMethod(writer http.ResponseWriter, request *http.Request) {
	// Parse device name, namespace and other information from api request
	urlItem := strings.Split(request.URL.Path, "/")
	deviceNamespace := urlItem[len(urlItem)-2]
	deviceName := urlItem[len(urlItem)-1]
	deviceID := parse.GetResourceID(deviceNamespace, deviceName)
	klog.V(2).Infof("Starting get all method of device %s in namespace %s.", deviceName, deviceNamespace)

	// Get all methods of the device from the devplane
	deviceMethodMap, propertyTypeMap, err := rs.devPanel.GetDeviceMethod(deviceID)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Get device method error: %v", err), http.StatusInternalServerError)
	} else {
		deviceMethod, err := rs.ParseMethodParameter(deviceMethodMap, propertyTypeMap, deviceName, deviceNamespace)
		if err != nil {
			http.Error(writer, fmt.Sprintf("Get device method error: %v", err), http.StatusInternalServerError)
			return
		}
		response := &DeviceMethodReadResponse{
			BaseResponse: NewBaseResponse(http.StatusOK),
			Data:         deviceMethod,
		}
		rs.sendResponse(writer, request, response, http.StatusOK)
		klog.V(2).Infof("Successfully obtained all methods of device %s", deviceName)
	}
}

// ParseMethodParameter add calling method, propertyName and property datatype to devicemethod parameter
func (rs *RestServer) ParseMethodParameter(deviceMethodMap map[string][]string, propertyTypeMap map[string]string, deviceName string, deviceNamespace string) (*common.DataMethod, error) {
	deviceMethod := common.DataMethod{
		Methods: make([]common.Method, 0),
	}
	for methodName, propertyList := range deviceMethodMap {
		method := common.Method{}
		method.Name = methodName
		method.Path = APIDeviceMethodRoute + "/" + deviceNamespace + "/" + deviceName + "/" + methodName + "/{propertyName}/{data}"
		parameter := make([]common.Parameter, 0)
		// get datatype of device property
		for _, propertyName := range propertyList {
			valueType, ok := propertyTypeMap[propertyName]
			if !ok {
				return nil, fmt.Errorf("unable to find device property %s defined in device method", propertyName)
			}
			parameter = append(parameter, common.Parameter{
				PropertyName: propertyName,
				ValueType:    valueType,
			})
		}
		method.Parameters = parameter
		deviceMethod.Methods = append(deviceMethod.Methods, method)
	}
	return &deviceMethod, nil
}

// DeviceWrite receive device method call request and complete data writing
func (rs *RestServer) DeviceWrite(writer http.ResponseWriter, request *http.Request) {
	// Parse device name, namespace and other information from api request
	urlItem := strings.Split(request.URL.Path, "/")
	deviceNamespace := urlItem[len(urlItem)-5]
	deviceName := urlItem[len(urlItem)-4]
	deviceMethodName := urlItem[len(urlItem)-3]
	propertyName := urlItem[len(urlItem)-2]
	data := urlItem[len(urlItem)-1]

	// Call device write command
	deviceID := parse.GetResourceID(deviceNamespace, deviceName)
	err := rs.devPanel.WriteDevice(deviceMethodName, deviceID, propertyName, data)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Write device data error: %v", err), http.StatusInternalServerError)
	} else {
		response := &DeviceWriteResponse{
			BaseResponse: NewBaseResponse(http.StatusOK),
			Message:      fmt.Sprintf("Write data %s to device %s successfully.", data, deviceID),
		}
		rs.sendResponse(writer, request, response, http.StatusOK)
	}
}

func (rs *RestServer) MetaGetModel(writer http.ResponseWriter, request *http.Request) {
	urlItem := strings.Split(request.URL.Path, "/")
	deviceNamespace := urlItem[len(urlItem)-2]
	deviceName := urlItem[len(urlItem)-1]
	deviceID := parse.GetResourceID(deviceNamespace, deviceName)
	device, err := rs.devPanel.GetDevice(deviceID)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Get device error: %v", err), http.StatusInternalServerError)
		return
	}
	driverInstancePtr := reflect.ValueOf(device)
	if driverInstancePtr.IsNil() {
		http.Error(writer, fmt.Sprintf("Get device error: %v", err), http.StatusInternalServerError)
		return
	}
	instance := driverInstancePtr.Elem().FieldByName("Instance")
	if instance.IsValid() {
		instance, ok := instance.Interface().(common.DeviceInstance)
		if ok {
			modelID := parse.GetResourceID(instance.Namespace, instance.Model)
			model, err := rs.devPanel.GetModel(modelID)
			if err != nil {
				http.Error(writer, fmt.Sprintf("Get device model error: %v", err), http.StatusInternalServerError)
			}
			response := &MetaGetModelResponse{
				BaseResponse: NewBaseResponse(http.StatusOK),
				DeviceModel:  &model,
			}
			rs.sendResponse(writer, request, response, http.StatusOK)
		} else {
			http.Error(writer, fmt.Sprintf("Get device instance error: %v", err), http.StatusInternalServerError)
		}
	} else {
		http.Error(writer, fmt.Sprintf("Get device instance error: %v", err), http.StatusInternalServerError)
	}
}

func (rs *RestServer) DataBaseGetDataByID(writer http.ResponseWriter, request *http.Request) {
	if rs.databaseClient == nil {
		http.Error(writer, "The database is not enabled. Please configure mapper and try again", http.StatusServiceUnavailable)
		return
	}
	response := &DataBaseResponse{
		BaseResponse: NewBaseResponse(http.StatusOK),
		Data:         nil,
	}
	rs.sendResponse(writer, request, response, http.StatusOK)
}
