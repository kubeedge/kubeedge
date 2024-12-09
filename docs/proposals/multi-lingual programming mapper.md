---
title: Multi-lingual Programming Mapper
status: implementable
authors:
  - "@ryWangkkk"
  - "@JiaweiGithub" 
approvers:
  - 
creation-date: 2024-08-01
last-updated: 2024-10-27
---
# Multi-lingual Programming Mapper

## Table of Contents
* [Introduction](#introduction)
* [Motivation](#motivation)
* [Goals](#goals)
* [Proposal](#proposal)
    * [Routine](#routine)
    * [Component](#component)
    * [Implementation](#implementation)
    * [Dependencies](#dependencies)

## Introduction
The Mapper module acts as a "translator" between KubeEdge and devices, 
enabling KubeEdge to interact with devices using various protocols,
retrieve device status, read necessary data from devices, and control edge devices.

## Motivation
For the current wide variety of protocols, not every protocol has an available library in Golang.
Additionally, some developers may have the need for custom protocols.
However, they might not be proficient in Go.
In contrast, they may prefer using Python, Java, or C/C++ to implement their own protocols.
Therefore, we need to improve the Mapper module to allow developers to choose their preferred programming language to implement custom communication protocols.

## Goals
Develop a Java version of the Mapper and implement the Modbus protocol as an example.
We will add versions in more languages later on, 
allowing users to choose the language they are proficient in to develop their own protocols.

## Proposal
### Routine
<img src="../images/proposals/multi-lingual programming mapper.png">

The routine of the Mapper is as follows:

1) The Mapper registers itself with EdgeCore using a **GrpcClient**.
2) The Mapper receives commands from EdgeCore through a **GrpcServer**, 
maintains the list of devices on the Mapper, and updates device expectations.
3) **DevicePanel** module creates a DevPanel to manage the device lifecycle.
4) **Driver module** reads and writes device data and converts it through the protocol.
5) **DevicePanel** module gets and sets data.
6) Meanwhile, **Data** module processes data in streaming or non-streaming modes,
7) and stores it in a database 
8) or pushes it to third-party apps.
9) or reports device status and data back to EdgeCore using **GrpcClient** module.

### Component
1) **Driver** defines a CustomizedClient to manage device access (initialization, reading, writing, and stopping the device). 
Its interface is defined as follows:

```
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
```

2) **Data** is responsible for handling data read from the Driver:

* Store regular data in databases such as InfluxDB2, MySQL, Redis, or TDengine.
* Process streaming data, such as video streams.
* Push data to third-party apps via HTTP or MQTT (if needed).

3) **DevicePanel** is responsible for managing the lifecycle of devices, 
such as starting, updating, deleting, and stopping devices, accessing information related to device and model, 
and handling twin data (reporting to EdgeCore, pushing to third-party applications, storing in databases, or sending to the Driver).
This is primarily implemented through the DevPanel class, with the following interface:

```
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
```

4) **Grpc** facilitates information exchange between the Mapper and EdgeCore via DMI.
This module is primarily divided into two parts:
* **GrpcClient** is responsible for registering the Mapper with EdgeCore and reporting device status, data, etc.

```
public class GrpcClient {
     public static Api.MapperRegisterResponse registerMapper(Config cfg, boolean withData) throws Exception {
        // registerMapper register mapper to EdgeCore,then get device and model list from edgecore.
        // if withData is true, edgecore will send device and model list.
        
        return response;
    }

    public static void reportDeviceStatus(Api.ReportDeviceStatusRequest request) throws InterruptedException {
        // reportDeviceStatus report device status to EdgeCore
    }
}
```

* **GrpcServer** is responsible for managing devices and models on the Mapper (e.g., registration, updating, removal) and receiving device expectations. 
According to the [api.proto](https://github.com/kubeedge/kubeedge/blob/master/pkg/apis/dmi/v1beta1/api.proto) file, the server should have the following functions:

```
public static abstract class DeviceMapperServiceImplBase implements io.grpc.BindableService {
    public void registerDevice(dmi.v1beta1.Api.RegisterDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RegisterDeviceResponse> responseObserver) throws Exception {
      asyncUnimplementedUnaryCall(getRegisterDeviceMethod(), responseObserver);
      // RegisterDevice registers a device to the mapper.
    }

    public void removeDevice(dmi.v1beta1.Api.RemoveDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getRemoveDeviceMethod(), responseObserver);
      // RemoveDevice unregisters a device to the device mapper.
    }

    public void updateDevice(dmi.v1beta1.Api.UpdateDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceResponse> responseObserver) throws Exception {
      asyncUnimplementedUnaryCall(getUpdateDeviceMethod(), responseObserver);
      // CreateDeviceModel creates a device model to the device mapper.
    }

    public void createDeviceModel(dmi.v1beta1.Api.CreateDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.CreateDeviceModelResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getCreateDeviceModelMethod(), responseObserver);
      // CreateDeviceModel creates a device model to the device mapper.
    }

    public void removeDeviceModel(dmi.v1beta1.Api.RemoveDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.RemoveDeviceModelResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getRemoveDeviceModelMethod(), responseObserver);
      // RemoveDeviceModel remove a device model to the device mapper.
    }

    public void updateDeviceModel(dmi.v1beta1.Api.UpdateDeviceModelRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.UpdateDeviceModelResponse> responseObserver) {
      asyncUnimplementedUnaryCall(getUpdateDeviceModelMethod(), responseObserver);
      // UpdateDeviceModel update a device model to the device mapper.
    }

    public void getDevice(dmi.v1beta1.Api.GetDeviceRequest request,
        io.grpc.stub.StreamObserver<dmi.v1beta1.Api.GetDeviceResponse> responseObserver) throws IOException {
      asyncUnimplementedUnaryCall(getGetDeviceMethod(), responseObserver);
      // GetDevice get the information of a device from the device mapper.
    }
```
5) **DMI** is responsible for implementing a Java version of the device manage interface.
As the interface is defined in [api.proto](https://github.com/kubeedge/kubeedge/blob/master/pkg/apis/dmi/v1beta1/api.proto),
and the ".proto" type file has the language-neutral nature,
we can use protobuf and protoc-gen-grpc-java to automatically generate the java version of the interface file easily.\
Consider keeping the development environment as close to the main project as possible,
we choose **protoc v3.19.4** and **protoc-gen-grpc-java v1.26.0**. 
We can place the api.proto in mapper/src/main/java/dmi/v1beta1, 
before automically generate java version file, we need make small change to the api.proto:
```
option go_package = "./;v1beta1";
option java_package = "dmi.v1beta1";// new added
package v1beta1;
```
After just adding a line, we can use the following command to generate the java version of the interface file:
```
protoc -I ".\src\main\java\dmi\" --java_out=src/main/java --grpc-java_out=src/main/java src/main/java/dmi/v1beta1/api.proto
```
6) **Http** provide API services, supporting directly obtaining device data from the device.
```
public class HTTPServer {
    public static class PingHandler implements HttpHandler {
        // handle /api/v1/ping request
    }
    public static class DeviceReadHandler implements HttpHandler{
        // handle /api/v1/device/{nameSpace}/{name}/{property} request
    }
    public static class MetaGetModelHandler implements HttpHandler{
        // handle /api/v1/meta/model/{nameSpace}/{name} request
    }
}
```
7) **Model, Service** is responsible for defining complex struct variables and interfaces separately.

### Implementation

```
mapper_default
├── src
│ └── main
│  ├── java
│  │ ├── Main.java ------------------ Main process
│  │ ├── config --------------------- Parse config files
│  │ ├── data ----------------------- Push data to 3rd app, save to database implementation layer
│  │ ├── devicepanel ---------------- Implementation of devicepanel layer, managing the device lifecycle
│  │ ├── driver --------------------- Device driver layer, reading and writing device data, then converts it through the customized protocol
│  │ ├── grpc ----------------------- Message interaction between Edgecore and mapper through DMI
│  │ ├── http ----------------------- Create HTTP server to provide API services, supporting directly obtaining device data from the device
│  │ ├── dmi ------------------------ Java version of device manage interface definition
│  │ ├── model ---------------------- Definition of complex variables
│  │ └── service -------------------- Definition of interfaces
│  └── resources
│    ├── logback.xml ---------------- Log configuration
│    ├── config.yaml ---------------- Global Configuration
├── hack
│ └── make-rules
│     ├── generate.sh
│     └── build.sh 
├── Dockerfile
├── Makefile
└── pom.xml
```

### Dependencies
java --version ≥ 11
protoc v3.19.4
protoc-gen-grpc-java v1.26.0
```
<!-- modbus -->
<dependency>
    <groupId>com.infiniteautomation</groupId>
    <artifactId>modbus4j</artifactId>
    <version>3.0.3</version>
</dependency>
<!-- grpc -->
<dependency>
  <groupId>io.grpc</groupId>
  <artifactId>grpc-netty-shaded</artifactId>
  <version>1.65.0</version>
  <scope>runtime</scope>
</dependency>
<dependency>
  <groupId>io.grpc</groupId>
  <artifactId>grpc-protobuf</artifactId>
  <version>1.65.0</version>
</dependency>
<dependency>
  <groupId>io.grpc</groupId>
  <artifactId>grpc-stub</artifactId>
  <version>1.65.0</version>
</dependency>
```