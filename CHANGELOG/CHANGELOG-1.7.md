  * [v1.7.1](#v171)
     * [Downloads for v1.7.1](#downloads-for-v171)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
     * [KubeEdge v1.7.1 Release Notes](#kubeedge-v171-release-notes)
        * [Changelog since v1.7.0](#changelog-since-v170) 
  * [v1.7.0](#v170)
     * [Downloads for v1.7.0](#downloads-for-v170)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
     * [KubeEdge v1.7 Release Notes](#kubeedge-v17-release-notes)
        * [1.7 What's New](#17-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


# v1.7.1

## Downloads for v1.7.1

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |


## KubeEdge v1.7.1 Release Notes

### Changelog since v1.7.0

- Fix nil pointer issue that caused cloudcore panic when restarting. ([#2876](https://github.com/kubeedge/kubeedge/pull/2876), [@muxuelan](https://github.com/muxuelan))


# v1.7.0

## Downloads for v1.7.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.7.0-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.7.0/kubeedge-v1.7.0-linux-arm64.tar.gz) |  61.4 MB | b40b3e8b6df54a8c00ee33dae66c2f515b7c76fb6850e0473a5697070a04559900903527936cdfbd913d5385251b9a979530d80c7bd033c05ef84a6f5219798f |
| [kubeedge-v1.7.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.7.0/kubeedge-v1.7.0-linux-arm.tar.gz) | 60.7 MB | b6d769f613888c0e083162b4845292b0d7cd87289109bb40973549d87c743ee69f106466b76fe1a6b9dd7ea38856ac99774de44e1d3111259c7c3c7c61472d3d |
| [kubeedge-v1.7.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.7.0/kubeedge-v1.7.0-linux-amd64.tar.gz) | 48 MB | 10fb1ad5950aae9802ed2e2368dfeb7924a6c4bfc4f52d159e22063d8264466a292845f84192eecd4e250abfed81d5597e644266a51eaf8a1f95d104ef38d1eb |


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.7.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.7.0/keadm-v1.7.0-linux-arm64.tar.gz) |  10.3 MB | 57fc98bbe6788418ee25d04a87592d90cdc61c7bfdeaa4e185d5735b71eeaead2f4a0608f195cd96bb5172284bfa72d081d5337b85b0f360d316a04183b3e3ba |
| [keadm-v1.7.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.7.0/keadm-v1.7.0-linux-arm.tar.gz) |  10.4 MB | d14fc0faf4031c9d7af463a0030f935ba3832cecf5e17d7f81fee2cfbaf93298eec3f802e2f4ac65a88acd673a1fc525e3a2dfc35f46157dffb072e7dcbecf68 |
| [keadm-v1.7.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.7.0/keadm-v1.7.0-linux-amd64.tar.gz) |  11.6 MB | cfa1001159bc5a4c44c5e67508cecc9e31bc9986da5469333559b207448dee1800e65bb53fdc948e78a6606dd8ccaa885e3d07029ee4f6e84ca14df49664dac3 |


## KubeEdge v1.7 Release Notes

### 1.7 What's New

**Active-Active HA Support of CloudCore for Large Scale Cluster [Alpha]**

CloudCore now supports Active-Active HA mode deployment, which provides better scalability support for large scale clusters.
Cloud-Edge tunnel can also work with multiple CloudCore instances.

Refer to the links for more details.
([#1560](https://github.com/kubeedge/kubeedge/issues/1560), [#2867](https://github.com/kubeedge/kubeedge/pull/2867))


**Support to manage Clusters on Edge [Alpha]**

In some scenarios, uses may have full-size Kubernetes clusters deployed on the edge.
With EdgeSite, users are now able to access clusters on edge (in private network, behind NATed gateway, etc) from center cloud.
([#2650](https://github.com/kubeedge/kubeedge/pull/2650), [#2858](https://github.com/kubeedge/kubeedge/pull/2858))


**Decoupled EdgeMesh from EdgeCore**

EdgeMesh aims to provide simplified network and services for edge applications.
The EdgeMesh module is now decoupled from EdgeCore and able to be deployed as an independent components in containers.

Refer to https://github.com/kubeedge/edgemesh for more details


**Mapper Framework**

Users are now able to use mapper framework to generate a new device mapper.
This simplifies the mapper development when users trying to integrate with new protocols or new devices.
([mappers-go#41](https://github.com/kubeedge/mappers-go/pull/41))


**Autonomic Kube-API Endpoint for Applications On Edge Nodes [Beta]**

Autonomic Kube-API Endpoint provides native Kubernetes API access on edge nodes.
It's very useful in cases users want to run third-party plugins and applications that depends on Kubernetes APIs on edge nodes.
With reliable message delivery and data autonomy provided by KubeEdge,
list-watch connections on edge nodes keep available even when nodes are located in high latency network or frequently get disconnected to the Cloud.

In this release, a bunch of corner case issues are fixed and the stability is improved. And the feature maturity is now Beta.


**Custom HTTP Request Routing between Cloud and Edge for Applications [Alpha]**

A new RuleEndpointType `servicebus` is added to RuleEndpoint API, to support custom http request routing between cloud and edge for applications. This simplifies the rest api access with http server on the edge while client is in the cloud.
 ([#2588](https://github.com/kubeedge/kubeedge/pull/2588))


### Important Steps before Upgrading

NA


### Other Notable Changes

- Implement update rule status ([#2594](https://github.com/kubeedge/kubeedge/pull/2594), [@MesaCrush](https://github.com/MesaCrush))
- Install crd for router in keadm ([#2608](https://github.com/kubeedge/kubeedge/pull/2608), [@fisherxu](https://github.com/fisherxu))
- Remove synckeeper in edgehub ([#2614](https://github.com/kubeedge/kubeedge/pull/2614), [@fisherxu](https://github.com/fisherxu))
- Shorten the reconnect wait time when connect failed ([#2641](https://github.com/kubeedge/kubeedge/pull/2641), [@fisherxu](https://github.com/fisherxu))
- upstream: refactor kubeClientGet ([#2694](https://github.com/kubeedge/kubeedge/pull/2694), [@zc2638](https://github.com/zc2638))
- cloud/dynamiccontroller: add ProcessApplication ([#2705](https://github.com/kubeedge/kubeedge/pull/2705), [@Iceber](https://github.com/Iceber))
- Add rules crd to clusterrole ([#2733](https://github.com/kubeedge/kubeedge/pull/2733), [@majoyz](https://github.com/majoyz))
- Disable image gc while ImageGCHighThreshold == 100 ([#2758](https://github.com/kubeedge/kubeedge/pull/2758), [@majoyz](https://github.com/majoyz))
- skip init edged if disable ([#2768](https://github.com/kubeedge/kubeedge/pull/2768), [@GsssC](https://github.com/GsssC))
- Remove mappers from kubeedge/kubeedge repo ([#2774](https://github.com/kubeedge/kubeedge/pull/2774), [@fisherxu](https://github.com/fisherxu))
- Add config of cloudcore token refresh frequence ([#2796](https://github.com/kubeedge/kubeedge/pull/2796), [@leofang94](https://github.com/leofang94))
- keadm: install CRDs corresponding to specific version ([#2803](https://github.com/kubeedge/kubeedge/pull/2803), [@daixiang0](https://github.com/daixiang0))
- make customsiz labels available when restart ([#2839](https://github.com/kubeedge/kubeedge/pull/2839), [@ttlv](https://github.com/ttlv))

### Bug Fixes

- fix keadm installation issue ([#2595](https://github.com/kubeedge/kubeedge/pull/2595), [@fisherxu](https://github.com/fisherxu))
- Fix the warning log when edgemesh is disabled ([#2599](https://github.com/kubeedge/kubeedge/pull/2599), [@hackers365](https://github.com/hackers365))
- fix cloudcore crash when nodekeepalivechannel is nil ([#2613](https://github.com/kubeedge/kubeedge/pull/2613), [@lvfei103650](https://github.com/lvfei103650))
- fix watch failed issue ([#2617](https://github.com/kubeedge/kubeedge/pull/2617), [@Abirdcfly](https://github.com/Abirdcfly))
- Fix image gc issue ([#2642](https://github.com/kubeedge/kubeedge/pull/2642), [@fisherxu](https://github.com/fisherxu))
- Fix container gc issue ([#2659](https://github.com/kubeedge/kubeedge/pull/2659), [@fisherxu](https://github.com/fisherxu))
- fix GetLocalIP IP lookup error ([#2689](https://github.com/kubeedge/kubeedge/pull/2689), [@zc2638](https://github.com/zc2638))
- cloud/dynamiccontroller: fix close application ([#2706](https://github.com/kubeedge/kubeedge/pull/2706), [@Iceber](https://github.com/Iceber))
- cloud/dynamiccontroller: fix toBytes ([#2707](https://github.com/kubeedge/kubeedge/pull/2707), [@Iceber](https://github.com/Iceber))
- edge/eventbus: fix pubCloudMsgToEdge ([#2726](https://github.com/kubeedge/kubeedge/pull/2726), [@Iceber](https://github.com/Iceber))
- has systemd double check ([#2734](https://github.com/kubeedge/kubeedge/pull/2734), [@k-9527](https://github.com/k-9527))
- Close response body when request done ([#2738](https://github.com/kubeedge/kubeedge/pull/2738), [@JackZxj](https://github.com/JackZxj))
- Stop to create listener when application center serve list request ([#2781](https://github.com/kubeedge/kubeedge/pull/2781), [@GsssC](https://github.com/GsssC))
- Fix: The server could not find the requested resource ([#2806](https://github.com/kubeedge/kubeedge/pull/2806), [@Rachel-Shao](https://github.com/Rachel-Shao))
- Bump k8s to 1.19.10 to fix metrics issue ([#2823](https://github.com/kubeedge/kubeedge/pull/2823), [@fisherxu](https://github.com/fisherxu))
