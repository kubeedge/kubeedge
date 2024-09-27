* [v1.15.4](#v1154)
    * [Downloads for v1.15.4](#downloads-for-v1154)
    * [KubeEdge v1.15.4 Release Notes](#kubeedge-v1154-release-notes)
        * [Changelog since v1.15.3](#changelog-since-v1153)
* [v1.15.3](#v1153)
    * [Downloads for v1.15.3](#downloads-for-v1153)
    * [KubeEdge v1.15.3 Release Notes](#kubeedge-v1153-release-notes)
        * [Changelog since v1.15.2](#changelog-since-v1152)
* [v1.15.2](#v1152)
    * [Downloads for v1.15.2](#downloads-for-v1152)
    * [KubeEdge v1.15.2 Release Notes](#kubeedge-v1152-release-notes)
        * [Changelog since v1.15.1](#changelog-since-v1151)
* [v1.15.1](#v1151)
    * [Downloads for v1.15.1](#downloads-for-v1151)
    * [KubeEdge v1.15.1 Release Notes](#kubeedge-v1151-release-notes)
        * [Changelog since v1.15.0](#changelog-since-v1150)
* [v1.15.0](#v1150)
    * [Downloads for v1.15.0](#downloads-for-v1150)
    * [KubeEdge v1.15 Release Notes](#kubeedge-v115-release-notes)
        * [1.15 What's New](#115-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)

# v1.15.4

## Downloads for v1.15.4

Download v1.15.4 in the [v1.15.4 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.15.4).

## KubeEdge v1.15.4 Release Notes

### Changelog since v1.15.3

- Fix parentID setting in func NewErrorMessage. ([#5735](https://github.com/kubeedge/kubeedge/pull/5735), [@luomengY](https://github.com/luomengY))
- Fix PersistentVolumes data stored at edge deleted abnormally.  ([#5888](https://github.com/kubeedge/kubeedge/pull/5888), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))


# v1.15.3

## Downloads for v1.15.3

Download v1.15.3 in the [v1.15.3 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.15.3).

## KubeEdge v1.15.3 Release Notes

### Changelog since v1.15.2

- Bump Kubernetes to the newest patch version v1.26.15. ([#5706](https://github.com/kubeedge/kubeedge/pull/5706), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix edgecore will not restart when the edge node cannot obtain the IP address. ([#5717](https://github.com/kubeedge/kubeedge/pull/5717), [@WillardHu](https://github.com/WillardHu))


# v1.15.2

## Downloads for v1.15.2

Download v1.15.2 in the [v1.15.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.15.2).

## KubeEdge v1.15.2 Release Notes

### Changelog since v1.15.1

- Fix default staticPodPath in windows. ([#5271](https://github.com/kubeedge/kubeedge/pull/5271), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix featuregates didn't take effect in edged. ([#5295](https://github.com/kubeedge/kubeedge/pull/5295), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix device status problem. ([#5336](https://github.com/kubeedge/kubeedge/pull/5336), [@wbc6080](https://github.com/wbc6080))
- Supports installing edgecore without installing the CNI plugin. ([#5367](https://github.com/kubeedge/kubeedge/pull/5367), [@luomengY](https://github.com/luomengY))


# v1.15.1

## Downloads for v1.15.1

Download v1.15.1 in the [v1.15.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.15.1).

## KubeEdge v1.15.1 Release Notes

### Changelog since v1.15.0

- Bump Kubernetes to the newest patch version 1.26.10. ([#5154](https://github.com/kubeedge/kubeedge/pull/5154), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix serviceaccount token not being deleted in edge DB. ([#5154](https://github.com/kubeedge/kubeedge/pull/5154), [#5199](https://github.com/kubeedge/kubeedge/pull/5199), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix Keadm upgrade if EdgeCore is stopped, the Keadm process will stop. ([#5111](https://github.com/kubeedge/kubeedge/pull/5111), [@wlq1212](https://github.com/wlq1212))
- Use ReportToCloud to determine whether to push device data from mapper to EdgeCore. ([#5116](https://github.com/kubeedge/kubeedge/pull/5116), [@luomengY](https://github.com/luomengY))
- Delete the historical version of CRD in cloudcore/CRD. ([#5147](https://github.com/kubeedge/kubeedge/pull/5147), [@luomengY](https://github.com/luomengY))
- Modify parameters for ginkgo v2. ([#5155](https://github.com/kubeedge/kubeedge/pull/5155), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix MetaServer panic with set StrictSerializer when handling create and update. ([#5183](https://github.com/kubeedge/kubeedge/pull/5183), [@Windrow14](https://github.com/Windrow14))
- Support building Windows-amd64 release for EdgeCore and Keadm. ([#5187](https://github.com/kubeedge/kubeedge/pull/5187), [@wujunyi792](https://github.com/wujunyi792))
- Remove unnecessary pid namespace config in copy-resource. ([#5191](https://github.com/kubeedge/kubeedge/pull/5191), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix null pointer error when PushMethod is not defined for device properties. ([#5204](https://github.com/kubeedge/kubeedge/pull/5204), [@luomengY](https://github.com/luomengY))
- Move pkg/util/grpcclient to pkg/grpcclient; RegisterMapper function should use pkg/grpcclient/config. ([#5208](https://github.com/kubeedge/kubeedge/pull/5208), [@cl2017](https://github.com/cl2017), [@wbc6080](https://github.com/wbc6080))
- Fix error logs when nodes repeatedly join different node groups. ([#5213](https://github.com/kubeedge/kubeedge/pull/5213), [@lishaokai1995](https://github.com/lishaokai1995), [@Onion-of-dreamed](https://github.com/Onion-of-dreamed))
- Resolve that users do not need to define the status module in device yaml.([#5217](https://github.com/kubeedge/kubeedge/pull/5217), [@luomengY](https://github.com/luomengY), [@wbc6080](https://github.com/wbc6080))
- Fix device model sync when add or delete devices. ([#5221](https://github.com/kubeedge/kubeedge/pull/5221), [@cl2017](https://github.com/cl2017), [@wbc6080](https://github.com/wbc6080))

# v1.15.0

## Downloads for v1.15.0

Download v1.15.0 in the [v1.15.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.15.0).

## KubeEdge v1.15 Release Notes

## 1.15 What's New

### Support Windows-based Edge Nodes

Edge computing involves various types of devices, including sensors, cameras, and industrial control devices,
some of which may run on the Windows OS. In order to support these devices and use cases, supporting Windows Server nodes
is necessary for KubeEdge.

In this release, KubeEdge supports the edge node running on Windows Server 2019, and supports Windows container running on edge node,
thereby extending KubeEdge to the Windows ecosystem and expanding its use cases and ecosystem.

Refer to the link for more details. ([#4914](https://github.com/kubeedge/kubeedge/pull/4914), [#4967](https://github.com/kubeedge/kubeedge/pull/4967))

### New v1beta1 version of Device API

The device API is updated from `v1alpha2` to `v1beta1`, in v1beta1 API updates include:

- The built-in protocols incude Modbus, Opc-UA and Bluetooth are removed in device instance, and the built-in mappers for these proytocols
still exists and will be maintained and updated to latest verison.
- Users must define the protocol config through `CustomizedValue` in `ProtocolConfig`.
- DMI date plane related fields are added, users can config the collection and reporting frequency of device data, and the destination
to whcih(such as database, httpserver) data is pushed.
- Controls whether to report device data to cloud.

Refer to the link for more details. ([#4983](https://github.com/kubeedge/kubeedge/pull/4983))


### Support Alpha version of DMI DatePlane and Mapper-Framework

Alpha version of DMI date plane is supported, DMI date plane is mainly implemented in mapper, providing interface for
pushing data, pulling data, and storing data in database.

To make writing mapper easier, a mapper development framework subproject **Mapper-Framework** is provided in this release.
Mapper-Framework provides mapper runtime libs and tools for scaffolding and code generation to bootstrap a new mapper project.
Users only need to run a command `make generate` to generate a mapper project, then add protocol related code to mapper.

Refer to the link for more details. ([#5023](https://github.com/kubeedge/kubeedge/pull/5023))


### Support Kubernetes native Static Pod on Edge Nodes
Kubernetes native `Static Pod` is supported on edge node in this release. Users can create pods on edge nodes by place pod manifests in
`/etc/kubeedge/manifests`, same as that on the Kubernetes node.

Refer to the link for more details. ([#4825](https://github.com/kubeedge/kubeedge/pull/4825))


### Support more Kubernetes Native Plugin Running on Edge Node

Kubernetes non-resource kind request `/version` is supported from edge node, users now can do `/version` requests in edge node from metaserver.
In addition, it can easily support other non-resource kind of requests like `/healthz` in edge node with the curent framework.
Many kubernetes plugins like cilium/calico which depend on these non-resource kind of requests, now can run on edge nodes.

Refer to the link for more details. ([#4904](https://github.com/kubeedge/kubeedge/pull/4904))

### Upgrade Kubernetes Dependency to v1.26.7

Upgrade the vendered kubernetes version to v1.26.7, users are now able to use the feature of new version on the cloud and on the edge side.

Refer to the link for more details. ([#4929](https://github.com/kubeedge/kubeedge/pull/4929))


## Important Steps before Upgrading

- In KubeEdge v1.15, new v1beta1 version of device API is incompatible with earlier versions of v1alpha1, users need to update the device API yamls to v1bata1 if you want to use v1.15.
- In KubeEdge v1.15, users need to upgrade the containerd to v1.6.0 or later. Containerd minor version 1.5 and older will not be supported in KubeEdge v1.15.
Ref: https://kubernetes.io/blog/2022/11/18/upcoming-changes-in-kubernetes-1-26/#cri-api-removal
- In KubeEdge v1.14, EdgeCore has removed the dockeshim support, so users can only use `remote` type runtime, and uses `containerd` runtime by default. If
you want to use `docker` runtime in v1.15, you also need to first set `edged.containerRuntime=remote` and corresponding docker configuration like `RemoteRuntimeEndpoint` and `RemoteImageEndpoint` in EdgeCore, then install the cri-dockerd tools as docs below:
https://github.com/kubeedge/kubeedge/issues/4843


