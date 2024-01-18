package httpserver

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

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
				common.WithValue(res),
				common.WithType(dataType),
			),
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
