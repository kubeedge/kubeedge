
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

## Other Notable Changes
- add custom image repo for keadm join beta ([#3654](https://github.com/kubeedge/kubeedge/pull/3654), [@TianTianBigWang](https://github.com/TianTianBigWang))
- keadm: beta join support remote runtime ([#3655](https://github.com/kubeedge/kubeedge/pull/3655), [@zc2638](https://github.com/zc2638))
- use sync mode to update pod status ([#3658](https://github.com/kubeedge/kubeedge/pull/3658), [@wackxu](https://github.com/wackxu))
- make log level configurable for local up kubeedge ([#3664](https://github.com/kubeedge/kubeedge/pull/3664), [@wackxu](https://github.com/wackxu))
- use dependency to pull images ([#3671](https://github.com/kubeedge/kubeedge/pull/3671), [@gy95](https://github.com/gy95))
- Move apis and client under kubeedge/cloud/pkg/ to kubeedge/pkg/ ([#3683](https://github.com/kubeedge/kubeedge/pull/3683), [@gy95](https://github.com/gy95))
- add subresource field in application for api with subresource ([#3693](https://github.com/kubeedge/kubeedge/pull/3693), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Add Keadm beta e2e ([#3699](https://github.com/kubeedge/kubeedge/pull/3699), [@zhu733756](https://github.com/zhu733756))
- keadm beta config images: support remote runtime ([#3700](https://github.com/kubeedge/kubeedge/pull/3700), [@gy95](https://github.com/gy95))
- use unified image management ([#3720](https://github.com/kubeedge/kubeedge/pull/3720), [@zc2638](https://github.com/zc2638))
- Use armhf as default for armv7/v6 ([#3723](https://github.com/kubeedge/kubeedge/pull/3723), [@fisherxu](https://github.com/fisherxu))
- add ErrStatus in api-server application ([#3742](https://github.com/kubeedge/kubeedge/pull/3742), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- support compile binaries with kubeedge/build-tools image ([#3756](https://github.com/kubeedge/kubeedge/pull/3756), [@gy95](https://github.com/gy95))
- add min TLS version for stream server ([#3764](https://github.com/kubeedge/kubeedge/pull/3764), [@snstaberah](https://github.com/snstaberah))
- Adding security policy ([#3778](https://github.com/kubeedge/kubeedge/pull/3778), [@vincentgoat](https://github.com/vincentgoat))
- chart: add cert domain config in helm chart ([#3802](https://github.com/kubeedge/kubeedge/pull/3802), [@lwabish](https://github.com/lwabish))
- add domain support for certgen.sh ([#3808](https://github.com/kubeedge/kubeedge/pull/3808), [@lwabish](https://github.com/lwabish))
- remove default KubeConfig for cloudcore ([#3836](https://github.com/kubeedge/kubeedge/pull/3836), [@wackxu](https://github.com/wackxu))
- Helm: Allow annotation of the cloudcore service ([#3856](https://github.com/kubeedge/kubeedge/pull/3856), [@ModeEngage](https://github.com/ModeEngage))
- add rate limiter for edgehub ([#3862](https://github.com/kubeedge/kubeedge/pull/3862), [@wackxu](https://github.com/wackxu))
- sync pod status immediately when status update ([#3891](https://github.com/kubeedge/kubeedge/pull/3891), [@wackxu](https://github.com/wackxu))


## Bug Fixes
- Fix readinessProbe and startupProbe not work ([#3665](https://github.com/kubeedge/kubeedge/pull/3665), [@wackxu](https://github.com/wackxu))
- Fix concurrent map iteration and map write bug ([#3670](https://github.com/kubeedge/kubeedge/pull/3670), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix local-up exit without error msg ([#3678](https://github.com/kubeedge/kubeedge/pull/3678), [@fisherxu](https://github.com/fisherxu))
- Fix and move helm base dir ([#3692](https://github.com/kubeedge/kubeedge/pull/3692), [@zhu733756](https://github.com/zhu733756))
- use runtime.GOARCH get machine arch instead of running shell command directly ([#3712](https://github.com/kubeedge/kubeedge/pull/3712), [@gy95](https://github.com/gy95))
- set deleteResource func param allowsOptions true ([#3730](https://github.com/kubeedge/kubeedge/pull/3730), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- when write hosts file we should use edged.dnsConfigure.ClusterDomain ([#3731](https://github.com/kubeedge/kubeedge/pull/3731), [@threestoneliu](https://github.com/threestoneliu))
- PodConfigMapsAndSecrets should handle Projected volume ([#3745](https://github.com/kubeedge/kubeedge/pull/3745), [@wackxu](https://github.com/wackxu))
- fix actualValue meta not set ([#3770](https://github.com/kubeedge/kubeedge/pull/3770), [@TianTianBigWang](https://github.com/TianTianBigWang))
- Fix rebuild build-tools image totally ([#3782](https://github.com/kubeedge/kubeedge/pull/3782), [@gy95](https://github.com/gy95))
- Fixed wrong format placeholder and nil warning on check ([#3790](https://github.com/kubeedge/kubeedge/pull/3790), [@jpanda-cn](https://github.com/jpanda-cn))
- fix endpointslices to kind EndpointSlice ([#3792](https://github.com/kubeedge/kubeedge/pull/3792), [@vincentgoat](https://github.com/vincentgoat))
- Pods configure imagePullSecret but doesn't exist in kubernetes cluster ([#3815](https://github.com/kubeedge/kubeedge/pull/3815), [@vincentgoat](https://github.com/vincentgoat))
- fix bug: fields.Selector for watch ([#3818](https://github.com/kubeedge/kubeedge/pull/3818), [@sdghchj](https://github.com/sdghchj))
- bugfix: rule.Status variable update ([#3821](https://github.com/kubeedge/kubeedge/pull/3821), [@cl2017](https://github.com/cl2017))
- fix Remove container failed ([#3826](https://github.com/kubeedge/kubeedge/pull/3826), [@gy95](https://github.com/gy95))
- fix keadm beta reset failed on edge node ([#3871](https://github.com/kubeedge/kubeedge/pull/3871), [@gy95](https://github.com/gy95))
- fix fuzzer extract message error ([#3899](https://github.com/kubeedge/kubeedge/pull/3899), [@vincentgoat](https://github.com/vincentgoat))
- fix type confusion in csi driver ([#3911](https://github.com/kubeedge/kubeedge/pull/3911), [@AdamKorcz](https://github.com/AdamKorcz))
