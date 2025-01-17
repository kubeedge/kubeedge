* [v1.17.5](#v1175)
    * [Downloads for v1.17.5](#downloads-for-v1175)
    * [KubeEdge v1.17.5 Release Notes](#kubeedge-v1175-release-notes)
        * [Changelog since v1.17.4](#changelog-since-v1174)
* [v1.17.4](#v1174)
    * [Downloads for v1.17.4](#downloads-for-v1174)
    * [KubeEdge v1.17.4 Release Notes](#kubeedge-v1174-release-notes)
        * [Changelog since v1.17.3](#changelog-since-v1173)
* [v1.17.3](#v1173)
    * [Downloads for v1.17.3](#downloads-for-v1173)
    * [KubeEdge v1.17.3 Release Notes](#kubeedge-v1173-release-notes)
        * [Changelog since v1.17.2](#changelog-since-v1172)
* [v1.17.2](#v1172)
    * [Downloads for v1.17.2](#downloads-for-v1172)
    * [KubeEdge v1.17.2 Release Notes](#kubeedge-v1172-release-notes)
        * [Changelog since v1.17.1](#changelog-since-v1171)
* [v1.17.1](#v1171)
    * [Downloads for v1.17.1](#downloads-for-v1171)
    * [KubeEdge v1.17.1 Release Notes](#kubeedge-v1171-release-notes)
        * [Changelog since v1.17.0](#changelog-since-v1170)
* [v1.17.0](#v1170)
    * [Downloads for v1.17.0](#downloads-for-v1170)
    * [KubeEdge v1.17 Release Notes](#kubeedge-v117-release-notes)
        * [1.17 What's New](#117-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)

# v1.17.5

## Downloads for v1.17.5

Download v1.17.5 in the [v1.17.5 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.17.5).

## KubeEdge v1.17.5 Release Notes

### Changelog since v1.17.4

- Fix clusterobjectsync cannot be deleted when edge node deleted. ([#6060](https://github.com/kubeedge/kubeedge/pull/6060), [@wbc6080](https://github.com/wbc6080))
- Fix duplicate generation of certificate if etcd fails. ([#6069](https://github.com/kubeedge/kubeedge/pull/6069), [@LRaito](https://github.com/LRaito))
- Fix iptablesmanager cannot clean iptables rules when CloudCore deleted. ([#6071](https://github.com/kubeedge/kubeedge/pull/6071), [@wbc6080](https://github.com/wbc6080))


# v1.17.4

## Downloads for v1.17.4

Download v1.17.4 in the [v1.17.4 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.17.4).

## KubeEdge v1.17.4 Release Notes

### Changelog since v1.17.3

- Fix errors due to singular and plural conversion in MetaServer. ([#5918](https://github.com/kubeedge/kubeedge/pull/5918), [@wbc6080](https://github.com/wbc6080))
- Fix install EdgeCore failed with CRI-O(>v1.29.2) for uid missing. ([#6014](https://github.com/kubeedge/kubeedge/pull/6014), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

# v1.17.3

## Downloads for v1.17.3

Download v1.17.3 in the [v1.17.3 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.17.3).

## KubeEdge v1.17.3 Release Notes

### Changelog since v1.17.2

- Optimize time format to support international time. ([#5820](https://github.com/kubeedge/kubeedge/pull/5820), [@WillardHu](https://github.com/WillardHu))
- Fix keadm reset lack of flag remote-runtime-endpoint. ([#5849](https://github.com/kubeedge/kubeedge/pull/5849), [@tangming1996](https://github.com/tangming1996))
- Fix PersistentVolumes data stored at edge deleted abnormally.  ([#5886](https://github.com/kubeedge/kubeedge/pull/5886), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

# v1.17.2

## Downloads for v1.17.2

Download v1.17.2 in the [v1.17.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.17.2).

## KubeEdge v1.17.2 Release Notes

### Changelog since v1.17.1

- Fix device mapper data collecting cycle calculation error. ([#5732](https://github.com/kubeedge/kubeedge/pull/5732), [@tangming1996](https://github.com/tangming1996))
- Fix message parentID setting in func NewErrorMessage. ([#5733](https://github.com/kubeedge/kubeedge/pull/5733), [@luomengY](https://github.com/luomengY))
- Fix pod status not to be updated when edge node offline. ([#5740](https://github.com/kubeedge/kubeedge/pull/5740), [@luomengY](https://github.com/luomengY))


# v1.17.1

## Downloads for v1.17.1

Download v1.17.1 in the [v1.17.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.17.1).

## KubeEdge v1.17.1 Release Notes

### Changelog since v1.17.0

- Bump Kubernetes to the newest patch version v1.28.11. ([#5697](https://github.com/kubeedge/kubeedge/pull/5697), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix edgecore will not restart when the edge node cannot obtain the IP address. ([#5715](https://github.com/kubeedge/kubeedge/pull/5715), [@WillardHu](https://github.com/WillardHu))
- Fix compatible issue with keadm init/upgrade --profile. ([#5718](https://github.com/kubeedge/kubeedge/pull/5718), [@luomengY](https://github.com/luomengY))



# v1.17.0

## Downloads for v1.17.0

Download v1.17.0 in the [v1.17.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.17.0).

## KubeEdge v1.17 Release Notes

## 1.17 What's New

### Support Edge Pods Using InClusterConfig to Access Kube-APIServer

The `InClusterConfig` mechanism of Kubernetes enables pods at cloud to directly access the Kube-APIServer. In this release, KubeEdge supports edge pods using `InClusterConfig` mechanism to access Kube-APIServer directly even through edge and cloud are in different network environment.

Refer to the link for more details. ([#5524](https://github.com/kubeedge/kubeedge/pull/5524), [#5541](https://github.com/kubeedge/kubeedge/pull/5541))

### Mapper Supports Video Streaming Data Reporting

Since Mapper could only process structured device data before, v1.17 adds video stream data processing features to Mapper-Framework.

- Edge camera device management

    v1.17 provides a built-in Mapper based on the Onvif protocol, which can manage Onvif network camera devices into the KubeEdge cluster and obtain the camera's authentication file and RTSP video stream.

- Video stream data processing

    In v1.17, stream data processing capabilities have been added to the Mapper-Framework data plane. The video stream reported by the edge camera device can be saved as frame files or video files.

Refer to the link for more details. ([#5448](https://github.com/kubeedge/kubeedge/pull/5448), [#5514](https://github.com/kubeedge/kubeedge/pull/5514), [mappers-go/#127](https://github.com/kubeedge/mappers-go/pull/127))

### Support Auto-Restarting for Edge Modules 

EdgeCore modules could fail to start due to non-configurable and recoverable matters such as process starting order. For example, if EdgeCore starts before `containerd.socket` is ready, edged fails to run kubelet and leads to EdgeCore exits.

In v1.17, we improve the BeeHive framework to support restarting modules. Users now can set the module in KubeEdge to automatically restart instead of restarting the whole component.

Refer to the link for more details. ([#5509](https://github.com/kubeedge/kubeedge/pull/5509), [#5513](https://github.com/kubeedge/kubeedge/pull/5513))

### Introduce `keadm ctl` Command to Support Pods Query and Restart at Edge

In v1.17, new command `keadm ctl` is introduced. Users can query and restart pods on edge nodes when they are offline through it.

- Query: `keadm ctl get pod [flags]`

- Restart:  `keadm ctl restart pod [flags]`

Refer to the link for more details. ([#5504](https://github.com/kubeedge/kubeedge/pull/5504))

### Keadm’s Enhancement

Some enhancements were made to the installation tool `keadm`:

- Refactor the command `keadm init`;
- Change the command `keadm generate` to `keadm manifest`;
- Add a flag `image-repository` for `keadm join` to support customize;
- Split the  `keadm reset` command into  `keadm reset cloud` and  `keadm reset edge`.

Refer to the link for more details. ([#5317](https://github.com/kubeedge/kubeedge/pull/5317))

### Add MySQL to Mapper Framework

In the pushMethod of the data plane in the mapper framework, a MySQL database source has been added. In DeviceInstance, basic configuration parameters for the MySQL client need to be added when user using.

Refer to the link for more details. ([#5376](https://github.com/kubeedge/kubeedge/pull/5376))

### Upgrade Kubernetes Dependency to v1.28.6 

Upgrade the vendered kubernetes version to v1.28.6, users are now able to use the feature of new version on the cloud and on the edge side.

Refer to the link for more details. ([#5412](https://github.com/kubeedge/kubeedge/pull/5412))

## Important Steps before Upgrading

- If you need to use the `InClusterConfig` feature for edge pods, you need to enable the switches of metaServer and dynamicController, and set featureGates `requireAuthorization=true` in the configuration files of both CloudCore and EdgeCore.

- If you want to use the `Auto-Restarting for Edge Modules` feature, you must enable the `moduleRestart` feature from the FeatureGates in EdgeCore.