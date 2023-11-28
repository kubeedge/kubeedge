
* [v1.15.0](#v1150)
    * [Downloads for v1.15.0](#downloads-for-v1150)
    * [KubeEdge v1.15 Release Notes](#kubeedge-v115-release-notes)
        * [1.15 What's New](#115-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)



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


