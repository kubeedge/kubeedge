package devicepanel;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hubspot.jackson.datatype.protobuf.ProtobufModule;
import dmi.v1beta1.Api;
import driver.CustomizedClient;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import model.CustomizedDev;
import driver.CustomizedProtocolConfig;
import model.DeviceInstance;
import model.DeviceModel;
import service.DevPanel_I;

import java.io.IOException;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.*;
import java.util.concurrent.locks.ReentrantLock;

import static devicepanel.Device.*;


@Slf4j
@Getter @Setter
public class DevPanel implements DevPanel_I {
    private Map<String, CustomizedDev> devices = new HashMap<>();
    private Map<String, DeviceModel> models = new HashMap<>();

    private Map<String, List<ScheduledFuture<?>>> deviceFutures = new HashMap<>();
    private CountDownLatch countDownLatch;
    private ReentrantLock serviceMutex = new ReentrantLock();

    @Override
    public void devInit(List<Api.Device> deviceList, List<Api.DeviceModel> deviceModelList) throws Exception {
        // init the devicePanel: build device and model list
        if (!deviceList.isEmpty() && !deviceModelList.isEmpty()) {

            // add models to devPanel
            for (Api.DeviceModel apiDeviceModel : deviceModelList) {
                DeviceModel deviceModel = Device.buildDeviceModelFromApi(apiDeviceModel);
                this.models.put(deviceModel.getId(), deviceModel);
            }

            // add devices to devPanel
            for (Api.Device apiDevice : deviceList) {
                DeviceModel deviceModel = this.models.get(apiDevice.getNamespace()+"/"+apiDevice.getSpec().getDeviceModelReference());
                DeviceInstance.ProtocolConfig protocolConfig = Device.buildProtocolFromApi(apiDevice);
                DeviceInstance deviceInstance = Device.buildDeviceFromApi(apiDevice, deviceModel);
                deviceInstance.setProtocolConfig(protocolConfig);

                CustomizedDev customizedDev = new CustomizedDev();
                customizedDev.setDeviceInstance(deviceInstance);

                this.devices.put(deviceInstance.getId(), customizedDev);
            }

            this.countDownLatch = new CountDownLatch(this.devices.size());
        }
    }

    @Override
    public void devStart() {
        // Start all devices
        for (Map.Entry<String, CustomizedDev> entry : this.devices.entrySet()) {
            String deviceId = entry.getKey();
            this.start(deviceId);
        }

        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            // stop devices
            for (Map.Entry<String, CustomizedDev> entry : this.devices.entrySet()) {
                String id = entry.getKey();
                CustomizedDev customizedDev = entry.getValue();
                for (Future<?> future : this.deviceFutures.get(id)) {
                    future.cancel(true);
                }
                try {
                    customizedDev.getCustomizedClient().stopDevice();
                } catch (Exception e) {
                    log.error("Fail to stop device {}, err {}", id, e.getMessage(), e);
                }
            }
            log.info("Exit mapper");
        }));

        try {
            this.countDownLatch.await();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
        log.info("All tasks are completed, Mapper will be closed");
    }

    @Override
    public void start(String deviceId) {
        CustomizedDev customizedDev = this.devices.get(deviceId);
        ObjectMapper objectMapper = new ObjectMapper();
        objectMapper.registerModule(new ProtobufModule());
        try {
            CustomizedProtocolConfig protocolConfig = objectMapper.readValue(customizedDev.getDeviceInstance().getProtocolConfig().getConfigData(), CustomizedProtocolConfig.class);

            // Initialize the client
            CustomizedClient customizedClient = new CustomizedClient(protocolConfig);
            customizedDev.setCustomizedClient(customizedClient);

            customizedClient.initDevice();
        } catch (IOException e) {
            log.error("Init CustomizedClient err: {}", e.getMessage());
        }
        // start pushToEdgeCore, save to database and publish data to 3rd app through http or mqtt
        List<ScheduledFuture<?>> futures = dataHandler(customizedDev);
        this.deviceFutures.put(deviceId, futures);
    }

    @Override
    public CustomizedDev getDevice(String deviceID) throws IOException {
        this.serviceMutex.lock();
        try {
            return this.devices.get(deviceID);
        } finally {
            this.serviceMutex.unlock();
        }
    }

    @Override
    public DeviceModel getModel(String modelID) {
        this.serviceMutex.lock();
        try {
            return this.models.get(modelID);
        } finally {
            this.serviceMutex.unlock();
        }
    }

    @Override
    public void updateDev(DeviceInstance device, DeviceModel model) {
        this.serviceMutex.lock();
        String deviceId = device.getId();
        String modelId = model.getId();
        try {

            // Stop Old device
            if (this.devices.get(deviceId) != null) {
                this.stopDev(deviceId);
            }

            // Start new deivce
            this.devices.put(deviceId, new CustomizedDev());
            this.devices.get(deviceId).setDeviceInstance(device);

            this.models.put(modelId, model);
        } finally {
            this.serviceMutex.unlock();
            this.start(deviceId);
            // Update the CountDownLatch with the current number of devices
            this.countDownLatch = new CountDownLatch(this.devices.size());
        }
    }

    // stop device: 1) stop related tasks 2) stop device
    @Override
    public void stopDev(String deviceID) {
        CustomizedDev dev = this.devices.get(deviceID);
        for (Future<?> future : this.deviceFutures.get(deviceID)) {
            future.cancel(true);
        }
        dev.getCustomizedClient().stopDevice();
        this.countDownLatch.countDown();
    }

    // remove device: 1) stop related tasks 2) stop device 3) clear related info in devPanel
    @Override
    public void removeDevice(String deviceID) {
        this.serviceMutex.lock();
        try {
            this.stopDev(deviceID);
            this.devices.remove(deviceID);
            this.deviceFutures.remove(deviceID);
        } finally {
            this.serviceMutex.unlock();
        }
    }


    @Override
    public void updateModel(DeviceModel model) {
        this.serviceMutex.lock();
        try {
            this.models.put(model.getId(), model);
        } finally {
            this.serviceMutex.unlock();
        }
    }

    @Override
    public void removeModel(String modelId) {
        this.serviceMutex.lock();
        try {
            this.models.remove(modelId);
        } finally {
            this.serviceMutex.unlock();
        }
    }

    @Override
    public String[] getTwinResult(String deviceID, String twinName) throws IOException {
        // Get twin's value, data type and timestamp;
        String[] res = new String[3];
        this.serviceMutex.lock();
        try{
            for (DeviceInstance.Twin twin: this.devices.get(deviceID).getDeviceInstance().getTwins()){
                if (twin.getPropertyName().equals(twinName)){
                    res[0] = twin.getReported().getValue();
                    res[1] = twin.getReported().getMetadata().getType();
                    res[2] = twin.getReported().getMetadata().getTimestamp();
                    return res;
                }
            }
        }finally {
            this.serviceMutex.unlock();
        }
        return null;
    }

    @Override
    public void updateDevTwins(String deviceID, List<DeviceInstance.Twin> twins) {

    }

    @Override
    public byte[] dealDeviceTwinGet(String deviceID, String twinName) {
        return new byte[0];
    }
}
