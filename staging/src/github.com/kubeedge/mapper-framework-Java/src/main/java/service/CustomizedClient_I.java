package service;

import driver.VisitorConfig;

public interface CustomizedClient_I {
    void initDevice();
    // initialize the device

    Object getDeviceData(VisitorConfig visitorConfig);
    // Get device data and Convert it to standard format through CustomizedClient

    void setDeviceData(Object data, VisitorConfig visitorConfig);
    // Set device data to expected value

    void stopDevice();
    // Stop the device
}
