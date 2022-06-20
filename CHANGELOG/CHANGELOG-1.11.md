
* [v1.11.0](#v1110)
    * [Downloads for v1.11.0](#downloads-for-v1110)
    * [KubeEdge v1.11 Release Notes](#kubeedge-v111-release-notes)
        * [1.11 What's New](#111-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


    

# v1.11.0

## Downloads for v1.11.0

Download v1.11.0 in the [v1.11.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.11.0).

## KubeEdge v1.11 Release Notes

## 1.11 What's New

### Node Group Management

Users can deploy applications to several node groups without writing deployment for every group. Node group management helps users to:

- Manage nodes in groups
- Spread apps among node groups 
- Run different version of app instances in different node groups
- Limit service endpoints in the same location as the client

Introduced two new APIs below to implement Node Group Management.

- **NodeGroup API**: represents a group of nodes that have the same labels.
- **EdgeApplication API**: contains the template of the application orgainzed by node groups, and the information of how to deploy different editions of the application to different node groups.

Refer to the links for more details.
([#3574](https://github.com/kubeedge/kubeedge/pull/3574), [#3719](https://github.com/kubeedge/kubeedge/pull/3719))


### Mapper SDK

Mapper-sdk is a basic framework written in go. Based on this framework, developers can more easily implement a new mapper.
Mapper-sdk has realized the connection to kubeedge, provides data conversion, and manages the basic properties and status of devices, etc. 
Basic capabilities and abstract definition of the driver interface. Developers only need to implement the 
customized protocol driver interface of the corresponding device to realize the function of mapper.

Refer to the links for more details.
([#70](https://github.com/kubeedge/mappers-go/pull/70))



### Beta sub-commands in Keadm to GA

Some new sub-commands in keadm move to GA, including containerized deployment, offline installation, etc.
Original `init` and `join` behaviors are replaced by implementation from `beta init` and `beta join`:
- CloudCore will be running in containers and managed by Kubernetes Deployment by default.
- keadm now downloads releases that packed as container image to edge nodes for node setup.

- `init`: CloudCore Helm Chart is integrated in init, which can be used to deploy containerized CloudCore.

- `join`: Installing edgecore as system service from docker image, no need to download from github release.

- `reset`: Reset the node, clean up the resources installed on the node by `init` or `join`. It will automatically detect the type of node to clean up.

- `manifest generate`: Generate all the manifests to deploy the cloudside components.


Refer to the links for more details.
([#3900](https://github.com/kubeedge/kubeedge/pull/3900))

### Deprecation of original `init` and `join`

Original `init` and `join` are deprecated, they have problems with offline installation, etc.

Refer to the links for more details.
([#3900](https://github.com/kubeedge/kubeedge/pull/3900))

### Next-gen Edged to Beta: Suitable for more scenarios

New version of the lightweight engine Edged, optimized from kubelet and integrated in edgecore, move to Beta.
New Edged will still communicate with the cloud through the reliable transmission tunnel.

Refer to the links for more details.
(Dev-Branch for beta: [feature-new-edged](https://github.com/kubeedge/kubeedge/tree/feature-new-edged))


## Important Steps before Upgrading

If you want to use keadm to deploy the KubeEdge v1.11.0, please note that the keadm `init` and `join` behaviors have been changed.

