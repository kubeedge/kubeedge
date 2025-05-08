---
title: Closed-Loop Flow Control via Topology Aware Routing
status: implementable
authors:
  - "@EraKin575"
  - "@tangming1996"
approvers:
  - "@WillardHu"
  - "@Shelley-BaoYue"
creation-date: 2024-11-16
last-updated: 2025-01-13
---

* [Introduction](#1-introduction)
* [Problem Statement](#2-problem-statement)
* [Proposed Solution](#3-proposed-solution)
  * [Integration of Topology Aware Routing for Closed-Loop Flow Control](#integration-of-topology-aware-routing-for-closed-loop-flow-control)
    * [Kubernetes Service Annotation](#1-kubernetes-service-annotation)
    * [Dynamic Endpoint Allocation with EndpointSlices](#2-dynamic-endpoint-allocation-with-endpointslices)
    * [Updates for Closed-Loop Traffic Management](#3-updates-for-closed-loop-traffic-management)
    * [Testing and Validation](#4-testing-and-validation)

# Solution Proposal: Closed-Loop Flow Control with Topology Aware Routing in Kubernetes

## 1. Introduction

**Closed-Loop Flow Control**:  
Currently, Kubernetes allows traffic routing through services, but there's a need for more refined control over traffic distribution, particularly in multi-zone environments. Closed-loop flow control seeks to provide a dynamic mechanism where traffic is continuously adjusted based on the current load, resource availability, and topological distribution of endpoints.

This proposal aims to enhance Kubernetes' traffic management with Topology Aware Routing to implement closed-loop flow control. By adjusting traffic flows dynamically, we can ensure that traffic stays within its origin zone, improving latency, throughput, and overall network performance, while also optimizing costs.

## 2. Problem Statement

1. **Static Traffic Routing**:  
   Traditional Kubernetes routing is static and doesn't take into account the dynamic load on each node or zone. This can lead to inefficient traffic distribution, with some zones being overloaded while others remain underutilized.

2. **Cross-Zone Traffic**:  
   Traffic often needs to traverse zones, leading to increased latency, potentially higher costs, and reduced performance. This dynamic issue becomes even more critical in clusters with varying node capacities.

3. **Closed-Loop Control**:  
   There is a lack of continuous, dynamic control over routing traffic within zones. We need an intelligent system that can react to current conditions, such as changing loads or resource availability, and adjust traffic flows accordingly.

## 3. Proposed Solution
![closed-loop-flow.png](..%2Fimages%2Fnode-group-management%2Fclosed-loop-flow.png)

### Integration of Topology Aware Routing for Closed-Loop Flow Control

To implement closed-loop flow control in Kubernetes, we propose using Topology Aware Routing along with custom heuristics to control traffic dynamically based on real-time metrics. This solution will involve multiple components:

#### 1. **Kubernetes Service Annotation**:
Enable topology-aware routing by setting the `service.kubernetes.io/topology-mode` annotation to `Auto`. This will dynamically populate zone-specific hints within the EndpointSlices, which kube-proxy can use to route traffic more efficiently.

**Example Service YAML with Topology Mode**:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    service.kubernetes.io/topology-mode: "Auto"
spec:
  selector:
    app: my-app
  ports:
    - port: 80
      targetPort: 8080
```
#### 2. **Dynamic Endpoint Allocation with EndpointSlices**:
The EndpointSlice controller will dynamically allocate endpoints to zones based on available resources and node capacity. It will consider factors such as CPU load and the number of available nodes in each zone to distribute traffic efficiently.
**Example EndpointSlice with Hints**:
```yaml
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: example-slice
  labels:
    kubernetes.io/service-name: my-service
addressType: IPv4
ports:
  - name: http
    protocol: TCP
    port: 80
endpoints:
  - addresses:
      - "10.1.2.3"
    conditions:
      ready: true
    hostname: pod-1
    zone: zone-a
```
#### 3. **Updates for Closed-Loop Traffic Management**:
To implement closed-loop flow control, the Endpoint Slices of each node in a nodegroup or equivalent controller will need to be updated to:
- Label nodes in a nodegroup with a label that indicates the zone it should be deployed too
- Use topology aware routing to confine traffic of nodes in a nodegroup to the zone they are labeled for
- Modify the EndpointSlice and Service objects to reflect updated routing information based on the current load and available resources.
#### 4. **Testing and Validation**:
The following tests will be conducted to ensure the success of this implementation:
- **Unit Tests**: Validate that the EndpointSlice controller correctly allocates endpoints based on topology and that traffic is routed based on the updated topology hints.
- **Integration Tests**: Verify that traffic is dynamically routed in response to changes in load and resource availability, ensuring that traffic stays within the same zone when possible.
- **Performance Testing**: Measure the impact on network latency and throughput before and after the implementation of topology-aware closed-loop flow control.