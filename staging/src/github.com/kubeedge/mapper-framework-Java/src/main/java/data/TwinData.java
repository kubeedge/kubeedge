package data;

import dmi.v1beta1.Api;
import driver.CustomizedClient;
import grpc.GrpcClient;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import driver.VisitorConfig;
import model.common.DeviceInstance;

import java.util.ArrayList;
import java.util.List;

import static data.DataConverter.convertToString;
@Slf4j
@Getter @Setter
public class TwinData {
    private String deviceName = "";
    private String deviceNameSpace = "";
    private CustomizedClient client;
    private String name = "";
    private DeviceInstance.TwinProperty observedDesired;
    private DeviceInstance.TwinProperty reported;
    private VisitorConfig visitorConfig;
    private long collectCycle;

    public void pushToEdgeCore(){
        Object actualValue = this.client.getDeviceData(this.visitorConfig);
        this.reported.setValue(convertToString(actualValue));
        this.reported.getMetadata().setTimestamp(String.valueOf(System.currentTimeMillis()));
        Api.DeviceStatus deviceStatus = Api.DeviceStatus.newBuilder()
                .addAllTwins(this.toApiTwinList()).build();
        Api.ReportDeviceStatusRequest reportDeviceStatusRequest = Api.ReportDeviceStatusRequest.newBuilder()
                .setDeviceName(this.deviceName)
                .setDeviceNamespace(this.deviceNameSpace)
                .setReportedDevice(deviceStatus)
                .build();
        try {
            GrpcClient.reportDeviceStatus(reportDeviceStatusRequest);
        } catch (InterruptedException ignored) {}
    }

    public List<Api.Twin> toApiTwinList(){
        List<Api.Twin> apiTwins = new ArrayList<>();
        Api.TwinProperty observedDesired = Api.TwinProperty.newBuilder()
                .setValue(this.observedDesired.getValue())
                .putMetadata("type",this.observedDesired.getMetadata().getType())
                .putMetadata("timestamp", this.observedDesired.getMetadata().getTimestamp())
                .build();

        Api.TwinProperty reported = Api.TwinProperty.newBuilder()
                .setValue(this.reported.getValue())
                .putMetadata("type",this.reported.getMetadata().getType())
                .putMetadata("timestamp", this.reported.getMetadata().getTimestamp())
                .build();

        Api.Twin apiTwin = Api.Twin.newBuilder()
                .setPropertyName(this.name)
                .setObservedDesired(observedDesired)
                .setReported(reported)
                .build();
        apiTwins.add(apiTwin);
        return apiTwins;
    }
}
