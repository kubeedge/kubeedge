package model;

import driver.CustomizedClient;
import lombok.Getter;
import lombok.Setter;
import model.common.DeviceInstance;

@Setter @Getter
public class CustomizedDev {
    private DeviceInstance deviceInstance;
    private CustomizedClient customizedClient;

    public CustomizedDev(DeviceInstance deviceInstance, CustomizedClient customizedClient) {
        this.deviceInstance = deviceInstance;
        this.customizedClient = customizedClient;
    }
    public CustomizedDev(){}
}
