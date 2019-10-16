# Roadmap

This document defines a high level roadmap for KubeEdge development.

The [milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) represent the most up-to-date plans.

KubeEdge 1.1 is our current stable branch. The roadmap below outlines new features that will be added to KubeEdge.

## 2019 Q4 Roadmap
- Support HA for cloudcore
- Support exec&logs API for edge application
- Support reliable message delivery from cloud to edge.
- Add protobuf support for data exchange format between cloud and edge
- Enhance performance and reliability of KubeEdge infrastructure.
- Support ingress at edge.
- Upgrade Kubernetes dependencies in vendor to v1.16.
- Improve contributor experience by defining project governance policies, release process, membership rules etc.
- Improve the performance and e2e tests with more metrics and scenarios.

## Future
- Support edge-cloud communication using edgemesh.
- Istio-based service mesh across Edge and Cloud where micro-services can communicate freely in the mesh.
- Enable function as a service at the Edge.
- Support more types of device protocols such as OPC-UA, Zigbee.
- Evaluate and enable much larger scale Edge clusters with thousands of Edge nodes and millions of devices.
- Enable intelligent scheduling of applications to large scale Edge clusters.
- Data management with support for ingestion of telemetry data and analytics at the edge.
- Security at the edge.
- Support for monitoring at the edge.
- Evaluate gRPC for cloud to edge communication.
