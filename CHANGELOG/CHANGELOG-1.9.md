
* [v1.9.2](#v192)
    * [Downloads for v1.9.2](#downloads-for-v192)
    * [KubeEdge v1.9.2 Release Notes](#kubeedge-v192-release-notes)
        * [Changelog since v1.9.1](#changelog-since-v191)
* [v1.9.1](#v191)
    * [Downloads for v1.9.1](#downloads-for-v191)
    * [KubeEdge v1.9.1 Release Notes](#kubeedge-v191-release-notes)
        * [Changelog since v1.9.0](#changelog-since-v190)
* [v1.9.0](#v190)
    * [Downloads for v1.9.0](#downloads-for-v190)
    * [KubeEdge v1.9 Release Notes](#kubeedge-v19-release-notes)
        * [1.9 What's New](#19-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


# v1.9.2

## Downloads for v1.9.2

Download v1.9.2 in the [v1.9.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.9.2).

## KubeEdge v1.9.2 Release Notes

### Changelog since v1.9.1

- Update current support K8s version to v1.21.4. ([#3486](https://github.com/kubeedge/kubeedge/pull/3486), [@gy95](https://github.com/gy95))

- Fix: change the resourceType of msg issued by synccontroller. ([#3496](https://github.com/kubeedge/kubeedge/pull/3496), [@Rachel-Shao](https://github.com/Rachel-Shao))

- Fix lots of duplicate logs "Failed to get obj" in objectsync.go. ([#3570](https://github.com/kubeedge/kubeedge/pull/3570), [@vincentgoat](https://github.com/vincentgoat))

- Filter pod by nodeName. ([#3594](https://github.com/kubeedge/kubeedge/pull/3594), [@yz271544](https://github.com/yz271544))

- Fix pv's objectsync spec.ObjectKind is empty. ([#3613](https://github.com/kubeedge/kubeedge/pull/3613), [@neiba](https://github.com/neiba))

- Fix readinessProbe and startupProbe bug. ([#3665](https://github.com/kubeedge/kubeedge/pull/3665), [@wackxu](https://github.com/wackxu))


# v1.9.1

## Downloads for v1.9.1

Download v1.9.1 in the [v1.9.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.9.1).

## KubeEdge v1.9.1 Release Notes

### Changelog since v1.9.0

- Fix crd created failed when install kubeedge using keadm. ([#3444](https://github.com/kubeedge/kubeedge/pull/3444), [@gy95](https://github.com/gy95))


# v1.9.0

## Downloads for v1.9.0

Download v1.9.0 in the [v1.9.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.9.0).

## KubeEdge v1.9 Release Notes

### 1.9 What's New


- **Support Custom HTTP Request Routing from Edge to Cloud through ServiceBus for Applications**

A HTTP server is added to ServiceBus, to support custom http request routing from edge to cloud
for applications. This simplifies the rest api access with http server on the cloud while client is in the edge.

Refer to the links for more details.
([#3254](https://github.com/kubeedge/kubeedge/issues/3254), [#3301](https://github.com/kubeedge/kubeedge/pull/3301))



- **Support CloudCore to run independently of the Kubernetes Master host**

CloudCore now supports to run independently of the Kubernetes Master host, iptablesmanager has been added as an independent
component, users only need to deploy the iptablesmanager to Kubernetes Master host, which now can
add the iptable rules for Cloud-Edge tunnel automatically

Refer to the links for more details.
([#3265](https://github.com/kubeedge/kubeedge/pull/3265))



- **EdgeMesh add tls and encryption security**

EdgeMesh's tunnel module adds tls and encryption security capabilities.
These features bring more secure protection measures to the user's edgemesh-server component and
reduce the risk of edgemesh-server being attacked.

Refer to the links for more details.
([EdgeMesh#127](https://github.com/kubeedge/edgemesh/pull/127))



- **Enhanced the ease of use of EdgeMesh**

EdgeMesh has many improvements in ease of use. Now users can easily deploy EdgeMesh's server and
agent components with a single command of helm. At the same time, the restriction on service port
naming is removed, and the docker0 dependency is removed, making it easier for users to use EdgeMesh.

Refer to the links for more details.
([EdgeMesh#123](https://github.com/kubeedge/edgemesh/pull/123), [EdgeMesh#126](https://github.com/kubeedge/edgemesh/pull/126), [EdgeMesh#136](https://github.com/kubeedge/edgemesh/pull/136), [EdgeMesh#175](https://github.com/kubeedge/edgemesh/pull/175         ))


- **Support containerized deployment of CloudCore using Helm**

CloudCore now supports containerized deployment using Helm, which provides better containerized deployment experience.

Refer to the links for more details.
([#3265](https://github.com/kubeedge/kubeedge/pull/3265))


- **Support compiled into rpm package and installed on OS such as openEuler using yum package manager**

KubeEdge now supports compiled into rpm package and installed on OS such as openEuler using yum package manager.

Refer to the links for more details.
([#3089](https://github.com/kubeedge/kubeedge/pull/3089), [#3171](https://github.com/kubeedge/kubeedge/pull/3171))


### Important Steps before Upgrading

If you want to deploy CloudCore independently of Kubernetes Master host, please deploy the independent iptablesmanager.

Refer to the links for more details.
([#3265](https://github.com/kubeedge/kubeedge/pull/3265))


### Other Notable Changes

- Rpminstaller: add support for openEuler ([#3089](https://github.com/kubeedge/kubeedge/pull/3089), [@CooperLi](https://github.com/CooperLi))
- Replaced 'kubeedge/pause' with multi arch image ([#3114](https://github.com/kubeedge/kubeedge/pull/3114), [@siredmar](https://github.com/siredmar))
- Make meta server addr configurable ([#3119](https://github.com/kubeedge/kubeedge/pull/3119), [@TianTianBigWang](https://github.com/TianTianBigWang))
- Added iptables to Dockerfile and made cloudcore privileged ([#3129](https://github.com/kubeedge/kubeedge/pull/3129), [@siredmar](https://github.com/siredmar))
- Added CustomInterfaceEnabled and CustomInterfaceName for edgecore ([#3130](https://github.com/kubeedge/kubeedge/pull/3130), [@siredmar](https://github.com/siredmar))
- Add experimental feature ([#3131](https://github.com/kubeedge/kubeedge/pull/3131), [@zc2638](https://github.com/zc2638))
- Feat(edge): node ephemeral storage info ([#3157](https://github.com/kubeedge/kubeedge/pull/3157), [@stingshen](https://github.com/stingshen))
- Support envFrom configmap in edge pods ([#3176](https://github.com/kubeedge/kubeedge/pull/3176), [@haoheipi](https://github.com/haoheipi))
- Update golang to 1.16 ([#3190](https://github.com/kubeedge/kubeedge/pull/3190), [@gy95](https://github.com/gy95))
- Metaserver: support shutdown server graceful  ([#3239](https://github.com/kubeedge/kubeedge/pull/3239), [@zc2638](https://github.com/zc2638))
- Support labelselector for metaserver ([#3262](https://github.com/kubeedge/kubeedge/pull/3262), [@chenchunxiu](https://github.com/chenchunxiu))


### Bug Fixes

- Fix dataProperty misplaced in devicecontroller ([#3065](https://github.com/kubeedge/kubeedge/pull/3065), [@waynechan9](https://github.com/waynechan9))
- Fix modbus slaveID cannot be 0 ([#3117](https://github.com/kubeedge/kubeedge/pull/3117), [@TianTianBigWang](https://github.com/TianTianBigWang))
- Enabled install debug handlers to enable the logs feature ([#3133](https://github.com/kubeedge/kubeedge/pull/3133), [@siredmar](https://github.com/siredmar))
- Idenfifying session not by IP of node ([#3136](https://github.com/kubeedge/kubeedge/pull/3136), [@siredmar](https://github.com/siredmar))
- Grant access to cloudcore for PV and PVC ([#3175](https://github.com/kubeedge/kubeedge/pull/3175), [@cuirunxing-hub](https://github.com/cuirunxing-hub))
- Fix(cloud): container exec occupies massive cpu  ([#3233](https://github.com/kubeedge/kubeedge/pull/3233), [@stingshen](https://github.com/stingshen))
- Fix LastTransitionTime update  ([#3235](https://github.com/kubeedge/kubeedge/pull/3235), [@Congrool](https://github.com/Congrool))
- Fix device does not exist in the downstream controller after it is added  ([#3257](https://github.com/kubeedge/kubeedge/pull/3257), [@TianTianBigWang](https://github.com/TianTianBigWang))
- Change kubeletEndpoint port when cloudstream conneted  ([#3277](https://github.com/kubeedge/kubeedge/pull/3277), [@chenchunxiu](https://github.com/chenchunxiu))
- Fix: resType does not match ([#3293](https://github.com/kubeedge/kubeedge/pull/3293), [@Rachel-Shao](https://github.com/Rachel-Shao))
- Fix edge node ip nil issue ([#3296](https://github.com/kubeedge/kubeedge/pull/3296), [@chenchunxiu](https://github.com/chenchunxiu))
- fix:Secret can't send to edge in SecretKeyRef ([#3329](https://github.com/kubeedge/kubeedge/pull/3329), [@QeelinDarly](https://github.com/QeelinDarly))
