# Roadmap

This document defines a high level roadmap for KubeEdge development.

The [milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) represent the most up-to-date plans.

The roadmap below outlines new features that will be added to KubeEdge.

## 2021 H1

### Core framework

#### Edge side list-watch
- Support list-watch interface at edge

#### Custom message transmission between cloud and edge
- Support transmission of custom message between cloud and edge

#### Support multi-instance cloudcore

#### Integration and verification of third-party CNI
- Flannel, Calico, etc.

#### Integration and verification of third-party CSI
- Rook, OpenEBS, etc.

#### Support managing clusters at edge from cloud (aka. EdgeSite)

#### Support ingress/gateway at edge.

### Maintainability

#### Deployment optimization
- Easier deployment
- Admission controller automated deployment

#### Automatic configuration of edge application offline migration time
- Modify Default tolerationSeconds Automatically

### IOT Device management

#### Device Mapper framework standard and framework generator
- Formulate mapper framework standard

#### Support mappers of more protocols
- OPC-UA mapper
- ONVIF mapper

### Security

#### Complete security vulnerability scanning


### Test

#### Improve the performance and e2e tests with more metrics and scenarios.


### Edge-cloud synergy AI

#### Supports KubeFlow/ONNX/Pytorch/Mindspore

#### Edge-cloud synergy training and inference


### MEC

#### Cross-edge cloud service discovery

#### 5G network capability exposure


## 2021 H2

### Core framework

#### Custom message transmission between cloud and edge
- Support CloudEvent protocol

#### Cross subnet communication of Data plane 
- Edge-edge cross subnet 
- Edge-cloud cross subnet

#### Unified Service Mesh support (Integrate with Istio/OSM etc.)

#### Cloud-edge synergy monitoring
- Provide support with prometheus push-gateway mode
- Data management with support for ingestion of telemetry data and analytics at the edge.

### IOT Device management

#### Device Mapper framework standard and framework generator
- Develop mapper framework generator

#### Support mappers of more protocols
- GB/T 28181 mapper

### Edge-cloud synergy AI

#### Intelligent edge benchmark


### MEC

#### Cloud-network convergence

#### Service catalog

#### Cross-edge cloud application roaming