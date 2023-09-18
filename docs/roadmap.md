# Roadmap

This document defines a high level roadmap for KubeEdge development.

The [milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) represent the most up-to-date plans.

The roadmap below outlines new features that will be added to KubeEdge.

## 2023 H2

### SIG Node

- Support edge nodes running on Windows.
- Capabilities enhancements for edge nodes, such as support static pods, event reporting and configurable application migration policies.
- Support edge nodes running on RTOS systems.
- Enhancements to device plugin, such as support for multiple virtual GPUs.
- Support for serverless computing.
- Feature of upgrading edge nodes from cloud move to GA.
- Optimization of node group features, such as support for more differentiated configuration parameters.

### SIG Device-IOT

- DMI data plane support (H1 has completed design).
- Migration solution for edge devices among multi-nodes based on DMI.
- Mapper framework support for DMI.
- Research on multi-language mappers support.
- Integration with time-series databases and other databases.
- Refactoring of Device and DeviceModel CRDs.
- Enhanced reliability for custom message transmission.

### SIG Security

- SLSA / CodeQL (There is still some provenance work remaining to reach SLSA L4).
- Spiffe Research.
- Support for certificates with multiple encryption algorithms, and provide interface capabilities.

### SIG Scalability

- Scalability and performance testing with EdgeMesh integrated.
- Scalability and performance testing for IoT devices scenario.

### Stability

- Stability maintenance of CloudCore, including stability testing and issue resolution.
- EdgeMesh stability.
- Enhanced reliability of cloud-edge collaboration, such as stability improvement of Edge Kube-API interface and logs/exec feature.

### SIG Testing

- Increase unit test coverage Improve.
- Improve e2e test case coverage (scenario-based coverage).
- Integration testing.
- Runtime and K8s version compatibility test.
- Keadm cross version compatibility test.
- Cloud-Edge cross version compatibility test.

### SIG Networking

- Node offline optimization
  - When a node goes offline, other nodes receive the update and remove the corresponding backend from the endpoint.
- Large-scale optimization
  - In large-scale deployments, there is a high load on the edge kube apiserver. Consider using IPVS (IP Virtual Server) technology to handle the requests efficiently.
  - Having a large number of services significantly increases the number of iptables rules on the nodes. Container Network supports CNI features.
- Performance optimization: Kernel-level traffic forwarding based on eBPF (extended Berkeley Packet Filter).
- Distributed messaging system.

### SIG Cluster-Lifecycle

- Support for Windows installation and deployment.
- Pre-download of images for edge applications.
- Router High Availability (HA) support.

### Docs

- Optimization of website documentation, including directory restructuring and improved comprehensiveness.
- Support for updating documentation versions.
- Completion of official website documentation, including the DMI developer guide, operational guide for monitoring and etc.
- Publish cases on website case studies.

### UI

- Dashboard release iteration.
- Add case studies on website.
- Add job center on website.
- Support for versioning on the website.

### SIG AI

- Support semi-automatic annotation in edge-cloud collaborative lifelong learning.
- Support runtime unseen task processing in edge-cloud collaborative lifelong learning.
- Support advanced offline unseen task processing in edge-cloud collaborative lifelong learning.

### SIG Robotics

- Add RoboDev Repository: Make it easier for developers to build robotic applications.
- Add RTF(ready to fly) Robotics E2E solutions: Teleoperation, RoboPilot.

### Experience 

- Example library enhancement
- Go online to Killer-Coda


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