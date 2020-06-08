# Roadmap

This document defines a high level roadmap for KubeEdge development.

The [milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) represent the most up-to-date plans.

KubeEdge 1.3 is our current stable branch. The roadmap below outlines new features that will be added to KubeEdge.

## 2020 Q2 Roadmap

- Support metrics-server in the cloud.
- Support Kubernetes exec API for edge application.
- Upgrade Kubernetes dependency to 1.18.
- Support edgenode certificate rotation.
- Upgrade golang to 1.14.
- Support ingress/gateway at edge.
- Device CRD improvement, support device protocol extension.
- Edge nodes cross subnet communication.
- Support list-watch from edgecore for applications on the edge.
- Collect data information sent from the edge side from CloudHub
- Improve KubeEdge installation experience
- Add more docs and move docs out of main repo


## Future

- Improve contributor experience by defining project governance policies, release process, membership rules etc.
- Improve the performance and e2e tests with more metrics and scenarios.
- Add protobuf support for data exchange format between cloud and edge
- Finish scalability test and publish report
- Support managing clusters at edge from cloud (aka. EdgeSite)
- Enhance performance and reliability of KubeEdge infrastructure.
- Support edge-cloud communication using edgemesh.
- Istio-based service mesh across Edge and Cloud where micro-services can communicate freely in the mesh.
- Enable function as a service at the Edge.
- Evaluate and enable much larger scale Edge clusters with thousands of Edge nodes and millions of devices.
- Enable intelligent scheduling of applications to large scale Edge clusters.
- Data management with support for ingestion of telemetry data and analytics at the edge.
- Security at the edge.
- Evaluate gRPC for cloud to edge communication.
