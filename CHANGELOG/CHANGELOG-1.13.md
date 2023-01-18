
* [v1.13.0](#v1130)
    * [Downloads for v1.13.0](#downloads-for-v1130)
    * [KubeEdge v1.13 Release Notes](#kubeedge-v113-release-notes)
        * [1.13 What's New](#113-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)



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

### EdgeMesh —— Added configurable field TunnelLimitConfig to edge-tunnel module

The tunnel stream of the edge-tunnel module is used to manage the data stream state of the tunnel. 
Users can obtain a stable and configurable tunnel stream to ensure the reliability of user application traffic forwarding.

Users can configure the cache size of tunnel stream according to `TunnelLimitConfig` to support larger application relay traffic.
Refer to the link for more details. ([#399](https://github.com/kubeedge/edgemesh/pull/399))

Cancel the restrictions on the relay to ensure the stability of the user's streaming application or long link application.
Refer to the link for more details. ([#400](https://github.com/kubeedge/edgemesh/pull/400))



## Important Steps before Upgrading

- EdgeCore now uses `containerd` runtime by default on KubeEdge v1.13. If you want to use `docker` runtime, you
  must set `edged.containerRuntime=docker` and corresponding docker configuration like `DockerEndpoint`, `RemoteRuntimeEndpoint` and `RemoteImageEndpoint` in EdgeCore.
