* [v1.13.5](#v1135)
    * [Downloads for v1.13.5](#downloads-for-v1135)
    * [KubeEdge v1.13.5 Release Notes](#kubeedge-v1135-release-notes)
        * [Changelog since v1.13.4](#changelog-since-v1134)
* [v1.13.4](#v1134)
    * [Downloads for v1.13.4](#downloads-for-v1134)
    * [KubeEdge v1.13.4 Release Notes](#kubeedge-v1134-release-notes)
        * [Changelog since v1.13.3](#changelog-since-v1133)
* [v1.13.3](#v1133)
    * [Downloads for v1.13.3](#downloads-for-v1133)
    * [KubeEdge v1.13.3 Release Notes](#kubeedge-v1133-release-notes)
        * [Changelog since v1.13.2](#changelog-since-v1132)
* [v1.13.2](#v1132)
    * [Downloads for v1.13.2](#downloads-for-v1132)
    * [KubeEdge v1.13.2 Release Notes](#kubeedge-v1132-release-notes)
        * [Changelog since v1.13.1](#changelog-since-v1131)
* [v1.13.1](#v1131)
    * [Downloads for v1.13.1](#downloads-for-v1131)
    * [KubeEdge v1.13.1 Release Notes](#kubeedge-v1131-release-notes)
        * [Changelog since v1.13.0](#changelog-since-v1130)
        * [Important Steps before Upgrading](#important-steps-before-upgrading-for-1131)
* [v1.13.0](#v1130)
    * [Downloads for v1.13.0](#downloads-for-v1130)
    * [KubeEdge v1.13 Release Notes](#kubeedge-v113-release-notes)
        * [1.13 What's New](#113-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)

# v1.13.5

## Downloads for v1.13.5

Download v1.13.5 in the [v1.13.5 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.13.5).

## KubeEdge v1.13.5 Release Notes

### Changelog since v1.13.4

- Fix featuregates didn't take effect in edged. ([#5297](https://github.com/kubeedge/kubeedge/pull/5297), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Supports installing edgecore without installing the CNI plugin. ([#5364](https://github.com/kubeedge/kubeedge/pull/5364), [@luomengY](https://github.com/luomengY))


# v1.13.4

## Downloads for v1.13.4

Download v1.13.4 in the [v1.13.4 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.13.4).

## KubeEdge v1.13.4 Release Notes

### Changelog since v1.13.3

- Resolve the deployment order dependency between mapper and device. ([#5149](https://github.com/kubeedge/kubeedge/pull/5149), [@luomengY](https://github.com/luomengY))
- Fix copy resources from the image throws nil runtimg error. ([#5188](https://github.com/kubeedge/kubeedge/pull/5188), [@WillardHu](https://github.com/WillardHu))
- Fix error logs when nodes repeatedly join different node groups. ([#5211](https://github.com/kubeedge/kubeedge/pull/5211), [@lishaokai1995](https://github.com/lishaokai1995), [@Onion-of-dreamed](https://github.com/Onion-of-dreamed))
- Bump Kubernetes to the newest patch version 1.23.17. ([#5224](https://github.com/kubeedge/kubeedge/pull/5224), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix serviceaccount token not being deleted in edge DB. ([#5224](https://github.com/kubeedge/kubeedge/pull/5224), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

# v1.13.3

## Downloads for v1.13.3

Download v1.13.3 in the [v1.13.3 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.13.3).

## KubeEdge v1.13.3 Release Notes

### Changelog since v1.13.2

- Fix upgrade time layout and lost time value issue. ([#5073](https://github.com/kubeedge/kubeedge/pull/5073), [@WillardHu](https://github.com/WillardHu))
- Fix start edgecore failed when using systemd cgroupdriver. ([#5102](https://github.com/kubeedge/kubeedge/pull/5102), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix remove pod cache failed. ([#5105](https://github.com/kubeedge/kubeedge/pull/5105), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix Keadm process stops abnormally when Keadm upgrade stops edgecore process. ([#5109](https://github.com/kubeedge/kubeedge/pull/5109), [@wlq1212](https://github.com/wlq1212))


# v1.13.2

## Downloads for v1.13.2

Download v1.13.2 in the [v1.13.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.13.2).

## KubeEdge v1.13.2 Release Notes

### Changelog since v1.13.1

- Fixed the kubeedge-version flag does not take effect in init and manifest generate command. ([#4936](https://github.com/kubeedge/kubeedge/pull/4936), [@WillardHu](https://github.com/WillardHu))
- Fix throws nil runtime error when decode AdmissionReview failed. ([#4971](https://github.com/kubeedge/kubeedge/pull/4971), [@WillardHu](https://github.com/WillardHu))
- Fix repeatedly reporting history device message to cloud. ([#4978](https://github.com/kubeedge/kubeedge/pull/4978), [@RyanZhaoXB](https://github.com/RyanZhaoXB))

# v1.13.1

## Downloads for v1.13.1

Download v1.13.1 in the [v1.13.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.13.1).

## KubeEdge v1.13.1 Release Notes

### Changelog since v1.13.0

- Fix MQTT container exited abnormally when edgecore using cri runtime. ([#4875](https://github.com/kubeedge/kubeedge/pull/4875), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Deal with error in delete pod upstream msg. ([#4878](https://github.com/kubeedge/kubeedge/pull/4878), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Update pod db when patch pod successfully. ([#4891](https://github.com/kubeedge/kubeedge/pull/4891), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Use nodeIP initialization in Kubelet, support reporting nodeIP dynamically . ([#4894](https://github.com/kubeedge/kubeedge/pull/4894), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix delete statefulset pod failed. ([#4873](https://github.com/kubeedge/kubeedge/pull/4873), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix container terminated when edgecore restart. ([#4870](https://github.com/kubeedge/kubeedge/pull/4870), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix iptables-manager and controller-manager image name. ([#4620](https://github.com/kubeedge/kubeedge/pull/4620), [@gy95](https://github.com/gy95))
- Wait for cache sync when cloudcore reboot. ([#4620](https://github.com/kubeedge/kubeedge/pull/4786), [@vincentgoat](https://github.com/vincentgoat))

### Important Steps before Upgrading for 1.13.1
- In previous versions, when edge node uses remote runtime (not docker runtime), using `keadm join` and specifying `--with-mqtt=true` to install edgecore will cause the Mosquitto container exits abnormally. In this release, this problem has been fixed. Users can specify `--with-mqtt=true` to start Mosquitto container when installing edgecore with `keadm join`.


# v1.13.0

## Downloads for v1.13.0

Download v1.13.0 in the [v1.13.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.13.0).

## KubeEdge v1.13 Release Notes

## 1.13 What's New

### Performance Improvement

- **CloudCore memory usage is reduced by 40%**, through unified generic Informer and reduce unnecessary cache. ([#4375](https://github.com/kubeedge/kubeedge/pull/4375)) ([#4377](https://github.com/kubeedge/kubeedge/pull/4377))
- List-watch dynamicController processing optimization, each watcher has a separate channel and goroutine processing to improve processing efficiency ([#4506](https://github.com/kubeedge/kubeedge/pull/4506))
- Added list-watch synchronization mechanism between cloud and edge and add dynamicController watch gc mechanism ([#4484](https://github.com/kubeedge/kubeedge/pull/4484))
- Removed 10s hard delay when offline nodes turn online ([#4490](https://github.com/kubeedge/kubeedge/pull/4490))
- Added prometheus monitor server and a metric connected_nodes to cloudHub. This metric tallies the number of connected nodes each cloudhub instance ([#3646](https://github.com/kubeedge/kubeedge/pull/3646))
- Added pprof for visualization and analysis of profiling data ([#3646](https://github.com/kubeedge/kubeedge/pull/3646))
- CloudCore configuration is now automatically adjusted according to nodeLimit to adapt to the number of nodes of different scales ([#4376](https://github.com/kubeedge/kubeedge/pull/4376))


### Security Improvement

- KubeEdge is proud to announce that we are digitally signing all release artifacts (including binary artifacts and container images). 
  Signing artifacts provides end users a chance to verify the integrity of the downloaded resource. It allows to mitigate man-in-the-middle attacks 
  directly on the client side and therefore ensures the trustfulness of the remote serving the artifacts. By doing this, we reached the 
  SLSA security assessment level L3 ([#4285](https://github.com/kubeedge/kubeedge/pull/4285))
- Remove the token field in the edge node configuration file edgecore.yaml to eliminate the risk of edge information leakage ([#4488](https://github.com/kubeedge/kubeedge/pull/4488))


### Upgrade Kubernetes Dependency to v1.23.15

Upgrade the vendered kubernetes version to v1.23.15, users are now able to use the feature of new version on the cloud and on the edge side.

Refer to the link for more details. ([#4509](https://github.com/kubeedge/kubeedge/pull/4509))


### Modbus Mapper based on DMI

Modbus Device Mapper based on DMI is provided, which is used to access Modbus protocol
devices and uses DMI to synchronize the management plane messages of devices with edgecore.

Refer to the link for more details. ([mappers-go#79](https://github.com/kubeedge/mappers-go/pull/79))


### Support Rolling Upgrade for Edge Nodes from Cloud

Users now able to trigger rolling upgrade for edge nodes from cloud, and specify number of concurrent upgrade nodes with `nodeupgradejob.spec.concurrency`. 
The default Concurrency value is 1, which means upgrade edge nodes one by one.
Refer to the link for more details. ([#4476](https://github.com/kubeedge/kubeedge/pull/4476))

### Test Runner for conformance test

KubeEdge has provided the runner of the conformance test, which contains the scripts 
and related files of the conformance test. 
Refer to the link for more details. ([#4411](https://github.com/kubeedge/kubeedge/pull/4411))

### EdgeMesh: Added configurable field TunnelLimitConfig to edge-tunnel module

The tunnel stream of the edge-tunnel module is used to manage the data stream state of the tunnel. 
Users can obtain a stable and configurable tunnel stream to ensure the reliability of user application traffic forwarding.

Users can configure the cache size of tunnel stream according to `TunnelLimitConfig` to support larger application relay traffic.
Refer to the link for more details. ([#399](https://github.com/kubeedge/edgemesh/pull/399))

Cancel the restrictions on the relay to ensure the stability of the user's streaming application or long link application.
Refer to the link for more details. ([#400](https://github.com/kubeedge/edgemesh/pull/400))



## Important Steps before Upgrading

- EdgeCore now uses `containerd` runtime by default on KubeEdge v1.13. If you want to use `docker` runtime, you
  must set `edged.containerRuntime=docker` and corresponding docker configuration like `DockerEndpoint`, `RemoteRuntimeEndpoint` and `RemoteImageEndpoint` in EdgeCore.
