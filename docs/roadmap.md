# Roadmap

This document defines a high level roadmap for KubeEdge development.

The [milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) represent the most up-to-date plans.

The roadmap below outlines new features that will be added to KubeEdge.

## 2023 H1 

### SIG Node

- Support WasmEdge integration on KubeEdge edgenode
- Support Kubectl attach to container running on edgenode
- Update Kubernetes dependency to v1.24.14, switch default container runtime at edge to containerd

### SIG Device-IOT

- Provide Modbus mapper based on DMI
- DMI Data plane
- Mapper framework support for DMI

### SIG Security

- Support authentication and authorization for Kube-API endpoint for applications on edge nodes
- Enhancements for edge plaintext storage, ensure tokens are not persisted on disk

### SIG Scalability

- Support cluster scope resource reliable delivery to edge nodes
- CloudCore memory usage is reduced by 40%, through unified generic informer and reduce unnecessary cache

### SIG Networking

- Add configurable field TunnelLimitConfig to edge-tunnel module
- EdgeMesh container network supports CNI features

### SIG AI

- Sedna
  - Support unstructured lifelong learning
  - Support unseen task recognition
  - Support displaying knowledge base

- Ianvs
  - Support lifelong learning throughout entire lifecycle
  - Provide classic lifelong learning testing metrics and support for visualizing test results
  - Provide real-world datasets and rich examples for lifelong learning testing

### SIG Testing

- Provide node conformance test suite
- Improve unit test coverage

### SIG Cluster-Lifecycle

- Provide a tool keink for running local KubeEdge clusters using Docker container “nodes”

### UI

- Alpha version of KubeEdge Dashboard
- Re-design KubeEdge website

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
