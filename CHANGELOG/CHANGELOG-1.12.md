
* [v1.12.0](#v1120)
    * [Downloads for v1.12.0](#downloads-for-v1120)
    * [KubeEdge v1.12 Release Notes](#kubeedge-v112-release-notes)
        * [1.12 What's New](#112-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)
    

# v1.12.0

## Downloads for v1.12.0

Download v1.12.0 in the [v1.12.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.12.0).

## KubeEdge v1.12 Release Notes

## 1.12 What's New

### Introducing Alpha Implementation of Next-gen Cloud Native Device Management Interface(DMI)

DMI makes KubeEdge's IoT device management more pluggable and modular in Cloud Native way,
which will cover Device Lifecycle Management, Device Operation, Device Data Management.

- **Device Lifecycle Management**: Making IOT device's lifecycle management as easy as managing a pod with simplifies operations
- **Device Operation**: Providing the ability to operate devices through Kubernetes API
- **Device Data Management**: Separate from device management, the data can be consumed by local application or sync to cloud in special tunnel 

Refer to the links for more details.
([#4013](https://github.com/kubeedge/kubeedge/pull/4013), [#3914](https://github.com/kubeedge/kubeedge/pull/3914))


### Next-gen Edged Graduates to GA: Suitable for more scenarios

New version of the lightweight engine Edged, optimized from kubelet and integrated in edgecore, move to GA.
New Edged will still communicate with the cloud through the reliable transmission tunnel.

Refer to the links for more details.
([#4184](https://github.com/kubeedge/kubeedge/pull/4184))

### Introducing High-Availability Mode for EdgeMesh

Compared with the previous centralized relay mode, EdgeMesh HA mode can set up multiple relay nodes.
When some relay nodes break down, other relay nodes can continue to provide relay services, which avoids single point of failure and greatly improves system stability.

In addition, a relay node that is too far away will cause a high latency. The HA relay node capability can provide intermediate nodes to shorten the latency.
The mDNS enables nodes in a LAN to communicate with each other without having to connect to an external network.

Refer to the links for more details. [EdgeMesh#372](https://github.com/kubeedge/edgemesh/pull/372)

### Support Edge Node Upgrade from Cloud

Introduce NodeUpgradeJob v1alpha1 API to upgrade edge nodes from cloud now. With NodeUpgradeJob API and Controller, users can:

- Using NodeUpgradeJob API to upgrade selected edge nodes from cloud 
- If upgrade fails, rollback to the original version

Refer to the links for more details.
([#4004](https://github.com/kubeedge/kubeedge/pull/4004), [#3822](https://github.com/kubeedge/kubeedge/pull/3822))


### Support Authorization for Edge Kube-API Endpoint

Authorization for Edge Kube-API Endpoint is now available. Third-party plugins and applications that depends on Kubernetes APIs on edge nodes
must use bearer token to talk to kube-apiserver via https server in MetaServer.

Refer to the links for more details.
([#4104](https://github.com/kubeedge/kubeedge/pull/4104), [#4226](https://github.com/kubeedge/kubeedge/pull/4226))


### New GigE Mapper

GigE Device Mapper with Golang implementation is provided, which is used to access GigE Vision protocol cameras.

Refer to the links for more details.
([mappers-go#72](https://github.com/kubeedge/mappers-go/pull/72))




## Important Steps before Upgrading

- If you want to use authorization for Edge Kube-API Endpoint, please enabled `RequireAuthorization` feature through feature gate both in CloudCore and EdgeCore. 
  If `RequireAuthorization` feature is enabled, metaServer will only serve for https request.
- If you want to upgrade edgemesh to v1.12, you do not need to deploy the existing edgemesh-server, and you need to configure relayNodes.
- If you want to run EdgeMesh v1.12 on KubeEdge v1.12, and use https request to talk to KubeEdge, you must set `kubeAPIConfig.metaServer.security.enable=true`.