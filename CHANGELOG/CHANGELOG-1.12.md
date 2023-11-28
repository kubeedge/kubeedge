* [v1.12.5](#v1125)
    * [Downloads for v1.12.5](#downloads-for-v1125)
    * [KubeEdge v1.12.5 Release Notes](#kubeedge-v1125-release-notes)
        * [Changelog since v1.12.4](#changelog-since-v1124)
* [v1.12.4](#v1124)
    * [Downloads for v1.12.4](#downloads-for-v1124)
    * [KubeEdge v1.12.4 Release Notes](#kubeedge-v1124-release-notes)
        * [Changelog since v1.12.3](#changelog-since-v1123)
* [v1.12.3](#v1123)
    * [Downloads for v1.12.3](#downloads-for-v1123)
    * [KubeEdge v1.12.3 Release Notes](#kubeedge-v1123-release-notes)
        * [Changelog since v1.12.2](#changelog-since-v1122)
        * [Important Steps before Upgrading](#important-steps-before-upgrading-for-1123)
* [v1.12.2](#v1122)
    * [Downloads for v1.12.2](#downloads-for-v1122)
    * [KubeEdge v1.12.2 Release Notes](#kubeedge-v1122-release-notes)
        * [Changelog since v1.12.1](#changelog-since-v1121)
* [v1.12.1](#v1121)
    * [Downloads for v1.12.1](#downloads-for-v1121)
    * [KubeEdge v1.12.1 Release Notes](#kubeedge-v1121-release-notes)
        * [Changelog since v1.12.0](#changelog-since-v1120)
* [v1.12.0](#v1120)
    * [Downloads for v1.12.0](#downloads-for-v1120)
    * [KubeEdge v1.12 Release Notes](#kubeedge-v112-release-notes)
        * [1.12 What's New](#112-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)

# v1.12.5

## Downloads for v1.12.5

Download v1.12.5 in the [v1.12.5 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.12.5).

## KubeEdge v1.12.5 Release Notes

### Changelog since v1.12.4

- Fix upgrade time layout and lost time value issue. ([#5072](https://github.com/kubeedge/kubeedge/pull/5072), [@WillardHu](https://github.com/WillardHu))
- Fix start edgecore failed when using systemd cgroupdriver. ([#5103](https://github.com/kubeedge/kubeedge/pull/5103), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix remove pod cache failed. ([#5106](https://github.com/kubeedge/kubeedge/pull/5106), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix Keadm process stops abnormally when Keadm upgrade stops edgecore process. ([#5108](https://github.com/kubeedge/kubeedge/pull/5108), [@wlq1212](https://github.com/wlq1212))
- Fix mqtt container would not start when using custom registry. ([#5101](https://github.com/kubeedge/kubeedge/pull/5101), [@WillardHu](https://github.com/WillardHu))

# v1.12.4

## Downloads for v1.12.4

Download v1.12.4 in the [v1.12.4 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.12.4).

## KubeEdge v1.12.4 Release Notes

### Changelog since v1.12.3

- Fixed the kubeedge-version flag does not take effect in init and manifest generate command. ([#4935](https://github.com/kubeedge/kubeedge/pull/4935), [@WillardHu](https://github.com/WillardHu))
- Fix throws nil runtime error when decode AdmissionReview failed. ([#4970](https://github.com/kubeedge/kubeedge/pull/4970), [@WillardHu](https://github.com/WillardHu))
- Fix repeatedly reporting history device message to cloud. ([#4979](https://github.com/kubeedge/kubeedge/pull/4979), [@RyanZhaoXB](https://github.com/RyanZhaoXB))

   
# v1.12.3

## Downloads for v1.12.3

Download v1.12.3 in the [v1.12.3 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.12.3).

## KubeEdge v1.12.3 Release Notes

### Changelog since v1.12.2

- Fix MQTT container exited abnormally when edgecore using cri runtime. ([#4876](https://github.com/kubeedge/kubeedge/pull/4876), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Deal with error in delete pod upstream msg. ([#4879](https://github.com/kubeedge/kubeedge/pull/4879), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Update pod db when patch pod successfully. ([#4892](https://github.com/kubeedge/kubeedge/pull/4892), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Use nodeIP initialization in Kubelet, support reporting nodeIP dynamically . ([#4893](https://github.com/kubeedge/kubeedge/pull/4893), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

### Important Steps before Upgrading for 1.12.3
- In previous versions, when edge node uses remote runtime (not docker runtime), using `keadm join` and specifying `--with-mqtt=true` to install edgecore will cause the Mosquitto container exits abnormally. In this release, this problem has been fixed. Users can specify `--with-mqtt=true` to start Mosquitto container when installing edgecore with `keadm join`.

# v1.12.2

## Downloads for v1.12.2

Download v1.12.2 in the [v1.12.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.12.2).

## KubeEdge v1.12.2 Release Notes

### Changelog since v1.12.1

- Fix prober not work in edgecore. ([#4572](https://github.com/kubeedge/kubeedge/pull/4572), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Optimize convert Kubelet flags, support more Kubelet flags. ([#4575](https://github.com/kubeedge/kubeedge/pull/4575), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix force delete pod when edgecore reconnect. ([#4596](https://github.com/kubeedge/kubeedge/pull/4596), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

# v1.12.1

## Downloads for v1.12.1

Download v1.12.1 in the [v1.12.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.12.1).

## KubeEdge v1.12.1 Release Notes

### Changelog since v1.12.0

- Fix binary edgecore incomplete during keadm join using remote-runtime ([#4320](https://github.com/kubeedge/kubeedge/pull/4320), [@gy95](https://github.com/gy95))
- keadm reset supports remote runtime remove mqtt container. ([#4322](https://github.com/kubeedge/kubeedge/pull/4322), [@gy95](https://github.com/gy95))
- bugfix patch operation in quic. ([#4340](https://github.com/kubeedge/kubeedge/pull/4340), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- cluster support quic and fix failed quic e2e. ([#4336](https://github.com/kubeedge/kubeedge/pull/4336), [@wackxu](https://github.com/wackxu))
- fix edgeconfig nodeLabels converted to kubelet nodeLabels. ([#4350](https://github.com/kubeedge/kubeedge/pull/4350), [@hexiaodai](https://github.com/hexiaodai))
- bugfix logs/exec. ([#4354](https://github.com/kubeedge/kubeedge/pull/4354), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- use objectsync to construct msg object when objectsync uid not equal ([#4370](https://github.com/kubeedge/kubeedge/pull/4370), [@neiba](https://github.com/neiba))



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

- If you want to upgrade KubeEdge to v1.12, the configuration file in EdgeCore has upgraded to v1alpha2, you must modify your configuration file of edged in EdgeCore to adapt the new edged.  
- If you want to use authorization for Edge Kube-API Endpoint, please enabled `RequireAuthorization` feature through feature gate both in CloudCore and EdgeCore. 
  If `RequireAuthorization` feature is enabled, metaServer will only serve for https request.
- If you want to upgrade edgemesh to v1.12, you do not need to deploy the existing edgemesh-server, and you need to configure relayNodes.
- If you want to run EdgeMesh v1.12 on KubeEdge v1.12, and use https request to talk to KubeEdge, you must set `kubeAPIConfig.metaServer.security.enable=true`.