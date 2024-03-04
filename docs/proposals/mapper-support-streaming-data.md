---
title: Mapper Support Streaming Data
authors:
- "@wbc6080"
  approvers:
  creation-date: 2024-3-5
  last-updated: 2024-3-5
  status: implementable
---

# Mapper Support Streaming Data

## Motivation

The data types currently collected by Mapper are discrete, such as `int`,`float` and other data types. 
However, edge devices that collect data in the form of streaming data are also common, such as surveillance cameras in cities. 
Mapper should enhance the management capabilities of streaming data devices.

## Goals

- Mapper can manage camera devices of edge nodes.
- Mapper is able to obtain video streams from camera devices in edge nodes.
- Mapper is able to forward video streams from edge to the cloud. (still in planning, may support in future)


## Design Details

### Architecture

<img src="../images/mapper/onvif-mapper.png">

### Camera Device Management Protocol

In the field of camera management, `ONVIF` is a commonly used protocol. `ONVIF` is a global and open industry forum, created a standard 
for how IP products within video surveillance and other physical security areas can communicate with each other. 
We can use the `ONVIF` protocol to complete features such as camera control and video analysis. The fields of the `ONVIF` protocol can be defined as

```yaml
protocol:
  customizedProtocol:
    protocolName: onvif
    configData:
      url: 192.168.168.64:80
      userName: admin
      password:  # passed by kubernetes secret form
```

In order to adapt to onvif device yaml, the corresponding structure needs to be added to the mapper device driver.

```go
type ProtocolConfig struct {
    ProtocolName string `json:"protocolName"`
    ConfigData   `json:"configData"`
}

type ConfigData struct {
    URL      string `json:"url,omitempty"` // the url of onvif device
    UserName string `json:"userName"`      // the username of onvif device
    Password string `json:"password"`      // the password of device user
}
```

We can define the IP address and port number of the edge camera device, and also need to define the username and password to access the camera device 
(the password can be mounted in the form of kubernetes secret to avoid clear text storage)

### Get Device Stream Data

According to the device management protocol, Mapper are able to connect and manage camera devices through third-party dependencies.
After that, Mapper can connect to the device based on the username and password, and obtain the camera `profileToken` and `RTSP` stream URI.
Generally, cameras use `RTSP` stream to provide video streaming services. Therefore, after obtaining the `RTSP` stream URI by Mapper, 
the video collected by the camera can be played through the video player on the edge node like VLC. 

## Plan

In version 1.17
- Provide a built-in Mapper of the onvif protocol based on Mapper-Framework
- Improve mapper-framework so that it can adapt to camera access with other protocols

In version 1.18
- Allow the cloud to obtain video streams collected by edge devices
- Allow device control through mapper (such as camera position information)
