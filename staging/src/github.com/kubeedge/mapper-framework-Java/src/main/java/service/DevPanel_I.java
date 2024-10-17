package service;

import dmi.v1beta1.Api;
import model.CustomizedDev;
import model.common.DeviceInstance;
import model.common.DeviceModel;

import java.io.IOException;
import java.util.List;

public interface DevPanel_I {
    void devInit(List<Api.Device> deviceList, List<Api.DeviceModel> deviceModelList) throws Exception;
    // devInit get device info to DevPanel by dmi interface
    void devStart();
    // devStart start devices to collect/push/save data to edgecore/app/database

    void start(String deviceID);

    // start the device
    CustomizedDev getDevice(String deviceID) throws IOException;
    // getDevice get device instance info
    void updateDev(DeviceInstance device, DeviceModel model);
    // updateDev stop old device, then update and start new device
    void stopDev(String deviceID);
    // stopDev stop device and the process
    void removeDevice(String deviceID);
    // removeDevice remove device instance
    DeviceModel getModel(String modelID);
    // getModel if the model exists, return device model
    void updateModel(DeviceModel model);
    // updateModel update device model
    void removeModel(String modelID);
    // removeModel remove device model
    String[] getTwinResult(String deviceID, String twinName) throws IOException;
    // getTwinResult Get twin's value and data type
    void updateDevTwins(String deviceID, List<DeviceInstance.Twin> twins);
    // updateDevTwins update device's twins
    byte[] dealDeviceTwinGet(String deviceID, String twinName);
    // dealDeviceTwinGet get device's twin data
}
