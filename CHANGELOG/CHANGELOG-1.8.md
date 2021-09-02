
  * [v1.8.0](#v180)
     * [Downloads for v1.8.0](#downloads-for-v180)
     * [KubeEdge v1.8 Release Notes](#kubeedge-v18-release-notes)
        * [1.8 What's New](#18-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


# v1.8.0

## Downloads for v1.8.0

Download v1.8.0 in the [v1.8.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.8.0).

## KubeEdge v1.8 Release Notes

### 1.8 What's New

**Active-Active HA Support of CloudCore for Large Scale Cluster [Beta]**

CloudCore now supports Active-Active HA mode deployment, which provides better scalability support for large scale clusters.
Cloud-Edge tunnel can also work with multiple CloudCore instances.
CloudCore now can add the iptable rules for Cloud-Edge tunnel automatically.

Refer to the links for more details.
([#1560](https://github.com/kubeedge/kubeedge/issues/1560), [#2999](https://github.com/kubeedge/kubeedge/pull/2999))


**EdgeMesh Architecture Modification**

EdgeMesh now has two parts: edgemesh-server and edgemesh-agent. The edgemesh-server requires a public IP address, when users use cross lan communication,
it can act as a relay server in the LibP2P mode or assist the agent to establish p2p hole punching.
The edgemesh-agent is used to proxy all application traffic of user nodes, acts as an agent for communication between pods
at different locations.

Refer to the links for more details.
([#19](https://github.com/kubeedge/edgemesh/pull/19))

**EdgeMesh Cross LAN Communication**

Users can use cross LAN communication feature to implement cross LAN edge to edge application communication and
cross LAN edge to cloud application communication.

Refer to the links for more details.
([#26](https://github.com/kubeedge/edgemesh/pull/26), [#37](https://github.com/kubeedge/edgemesh/pull/37), [#57](https://github.com/kubeedge/edgemesh/pull/57))


**Onvif Device Mapper**

Onvif Device Mapper with Golang implementation is provided, based on new Device Mapper Standard.
Users now can use onvif device mapper to manage the ONVIF IP camera.

Refer to the links for more details.
([mappers-go#48](https://github.com/kubeedge/mappers-go/pull/48))

**Kubernetes Dependencies Upgrade**

Upgrade the vendered kubernetes version to v1.21.4, users now can use the feature of new version
on the cloud and on the edge side.

Refer to the links for more details.
([#3021](https://github.com/kubeedge/kubeedge/pull/3021), [#3034](https://github.com/kubeedge/kubeedge/pull/3034))


### Important Steps before Upgrading

**NOTE:**
In v1.8 EdgeMesh has been decoupled from edgecore and moved to [edgemesh](https://github.com/kubeedge/edgemesh) repo, if you are using EdgeMesh,
Please install the latest version of edgemesh.

Refer to the links for more details.
([#2916](https://github.com/kubeedge/kubeedge/pull/2916))

### Other Notable Changes

- Refactor edgesite: import functions and structs instead of copying code ([#2893](https://github.com/kubeedge/kubeedge/pull/2893), [@liufen90](https://github.com/liufen90))
- Avoiding update cm after created a new cm ([#2913](https://github.com/kubeedge/kubeedge/pull/2913), [@huang339](https://github.com/huang339))
- Solved the checksum file download problem when ke was installed offline ([#2909](https://github.com/kubeedge/kubeedge/pull/2909), [@Rachel-Shao](https://github.com/Rachel-Shao))
- cloudcore support configmap dynamic update when the env of container inject from configmap or secret ([#2931](https://github.com/kubeedge/kubeedge/pull/2931), [@rzyeleven](https://github.com/rzyeleven))
- Remove edgemesh from edgecore ([#2916](https://github.com/kubeedge/kubeedge/pull/2916), [@fisherxu](https://github.com/fisherxu))
- keadm: support customsized labels when use join command ([#2827](https://github.com/kubeedge/kubeedge/pull/2827), [@ttlv](https://github.com/ttlv))
- support k8s v1.21.X ([#3021](https://github.com/kubeedge/kubeedge/pull/3021), [@gy95](https://github.com/gy95))
- Handling node/*/membership/detail ([#3025](https://github.com/kubeedge/kubeedge/pull/3025), [@subpathdev](https://github.com/subpathdev))
- sync the response message unconditionally ([#3014](https://github.com/kubeedge/kubeedge/pull/3014), [@sdghchj](https://github.com/sdghchj))
- support default NVIDIA SMI command ([#2680](https://github.com/kubeedge/kubeedge/pull/2680), [@zc2638](https://github.com/zc2638))

### Bug Fixes

- modify the value of tunnel port ([#2876](https://github.com/kubeedge/kubeedge/pull/2876), [@muxuelan](https://github.com/muxuelan))
- Fix message to apiserver ([#2883](https://github.com/kubeedge/kubeedge/pull/2883), [@qugq0228](https://github.com/qugq0228))
- fix incorrect use of TrimLeft or TrimRight ([#2907](https://github.com/kubeedge/kubeedge/pull/2907), [@gy95](https://github.com/gy95))
- cloudhub: fix signEdgeCert nil pointer ([#2935](https://github.com/kubeedge/kubeedge/pull/2935), [@zc2638](https://github.com/zc2638))
- use UpdateDeviceStatusWorkers as updateDeviceStatus routines ([#3024](https://github.com/kubeedge/kubeedge/pull/3024), [@gy95](https://github.com/gy95))
- Solve the concurrent map write for metaserver handler.go ([#2955](https://github.com/kubeedge/kubeedge/pull/2955), [@yz271544](https://github.com/yz271544))
- Fixed modbus config parameters null value invalid ([#3049](https://github.com/kubeedge/kubeedge/pull/3049), [@TianTianBigWang](https://github.com/TianTianBigWang))
- admission: fix pod toleration replace  ([#2848](https://github.com/kubeedge/kubeedge/pull/2848), [@zc2638](https://github.com/zc2638))
- fix target kubeletendpoint port in metrics request ([#3010](https://github.com/kubeedge/kubeedge/pull/3010), [@cuirunxing-hub](https://github.com/cuirunxing-hub))