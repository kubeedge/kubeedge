* [v1.14.3](#v1143)
    * [Downloads for v1.14.3](#downloads-for-v1143)
    * [KubeEdge v1.14.3 Release Notes](#kubeedge-v1143-release-notes)
        * [Changelog since v1.14.2](#changelog-since-v1142)
* [v1.14.2](#v1142)
    * [Downloads for v1.14.2](#downloads-for-v1142)
    * [KubeEdge v1.14.2 Release Notes](#kubeedge-v1142-release-notes)
        * [Changelog since v1.14.1](#changelog-since-v1141)
* [v1.14.1](#v1141)
    * [Downloads for v1.14.1](#downloads-for-v1141)
    * [KubeEdge v1.14.1 Release Notes](#kubeedge-v1141-release-notes)
        * [Changelog since v1.14.0](#changelog-since-v1140)
        * [Important Steps before Upgrading](#important-steps-before-upgrading-for-1141)
* [v1.14.0](#v1140)
    * [Downloads for v1.14.0](#downloads-for-v1140)
    * [KubeEdge v1.14 Release Notes](#kubeedge-v114-release-notes)
        * [1.14 What's New](#114-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)

# v1.14.3

## Downloads for v1.14.3

Download v1.14.3 in the [v1.14.3 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.14.3).

## KubeEdge v1.14.3 Release Notes

### Changelog since v1.14.2

- Fix upgrade time layout and lost time value issue. ([#5074](https://github.com/kubeedge/kubeedge/pull/5074), [@WillardHu](https://github.com/WillardHu))
- Fix start edgecore failed when using systemd cgroupdriver. ([#5076](https://github.com/kubeedge/kubeedge/pull/5076), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix remove pod cache failed. ([#5104](https://github.com/kubeedge/kubeedge/pull/5104), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix Keadm process stops abnormally when Keadm upgrade stops edgecore process. ([#5110](https://github.com/kubeedge/kubeedge/pull/5110), [@wlq1212](https://github.com/wlq1212))

# v1.14.2

## Downloads for v1.14.2

Download v1.14.2 in the [v1.14.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.14.2).

## KubeEdge v1.14.2 Release Notes

### Changelog since v1.14.1

- Fixed the cloudcore panic caused when using ClusterObjectSyncs resource. ([#4918](https://github.com/kubeedge/kubeedge/pull/4918), [@ZhengXinwei-F](https://github.com/ZhengXinwei-F))
- Fixed the kubeedge-version flag does not take effect in init and manifest generate command. ([#4937](https://github.com/kubeedge/kubeedge/pull/4937), [@WillardHu](https://github.com/WillardHu))
- Fix throws nil runtime error when decode AdmissionReview failed. ([#4972](https://github.com/kubeedge/kubeedge/pull/4972), [@WillardHu](https://github.com/WillardHu))
- Fix repeatedly reporting history device message to cloud. ([#4977](https://github.com/kubeedge/kubeedge/pull/4977), [@RyanZhaoXB](https://github.com/RyanZhaoXB))


# v1.14.1

## Downloads for v1.14.1

Download v1.14.1 in the [v1.14.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.14.1).

## KubeEdge v1.14.1 Release Notes

### Changelog since v1.14.0

- Fix MQTT container exited abnormally when edgecore using cri runtime. ([#4874](https://github.com/kubeedge/kubeedge/pull/4874), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Deal with error in delete pod upstream msg. ([#4877](https://github.com/kubeedge/kubeedge/pull/4877), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Update pod db when patch pod successfully. ([#4890](https://github.com/kubeedge/kubeedge/pull/4890), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Use nodeIP initialization in Kubelet, support reporting nodeIP dynamically . ([#4895](https://github.com/kubeedge/kubeedge/pull/4895), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix delete statefulset pod failed. ([#4872](https://github.com/kubeedge/kubeedge/pull/4872), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix container terminated when edgecore restart. ([#4871](https://github.com/kubeedge/kubeedge/pull/4871), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix parsing response message failed in metaclient. ([#4898](https://github.com/kubeedge/kubeedge/pull/4898), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

### Important Steps before Upgrading for 1.14.1
- In previous versions, when edge node uses remote runtime (not docker runtime), using `keadm join` and specifying `--with-mqtt=true` to install edgecore will cause the Mosquitto container exits abnormally. In this release, this problem has been fixed. Users can specify `--with-mqtt=true` to start Mosquitto container when installing edgecore with `keadm join`.


# v1.14.0

## Downloads for v1.14.0

Download v1.14.0 in the [v1.14.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.14.0).

## KubeEdge v1.14 Release Notes

## 1.14 What's New

### Support Authentication and Authorization for Kube-API Endpoint for Applications On Edge Nodes

The Kube-API endpoint for edge applications is implemented through MetaServer in edegcore. However, in previous versions, 
the authentication and authorization of Kube-API endpoint are performed in the cloud, which prevents authentication and authorization 
especially in offline scenarios on the edge node.

In this release, the authentication and authorization functionalities are implemented within the MetaServer at edge, which allows for 
limiting the access permissions of edge applications when accessing Kube-API endpoint at edge.

Refer to the link for more details. ([#4802](https://github.com/kubeedge/kubeedge/pull/4802))

### Support Cluster Scope Resource Reliable Delivery to Edge Node

The cluster scope resource can guarantee deliver to the edge side reliably since this release, 
especially include using list-watch global resources, the cluster scope resource can be delivered to the edge side reliably,
and the edge applications can work normally.

Refer to the link for more details. ([#4758](https://github.com/kubeedge/kubeedge/pull/4758))


### Upgrade Kubernetes Dependency to v1.24.14

Upgrade the vendered kubernetes version to v1.24.14, users are now able to use the feature of new version on the cloud and on the edge side.
**Note**: The dockershim has been removed, which means users can't use docker runtime directly in this release.

Refer to the link for more details. ([#4789](https://github.com/kubeedge/kubeedge/pull/4789))


### Support Kubectl Attach to Container Running on Edge Node

KubeEdge already support `kubectl logs/exe` command, `kubectl attach` is supported in this release.
`kubectl attach` command can attach to a running container at edge node.
Users can execute these commands in the cloud and no need to operate on the edge nodes.

Refer to the link for more details. ([#4734](https://github.com/kubeedge/kubeedge/pull/4734))


### Alpha version of KubeEdge Dashboard

KubeEdge dashboard provides a graphical user interface (GUI) for managing and monitoring your KubeEdge clusters. 
It allows users to manage edge applications running in the cluster and troubleshoot them.

Refer to the link for more details. (https://github.com/kubeedge/dashboard)



## Important Steps before Upgrading

- On KubeEdge v1.14, EdgeCore has removed the dockeshim support, so users can only use `remote` type runtime, and uses `containerd` runtime by default. If you want to use `docker` runtime, you
  must first set `edged.containerRuntime=remote` and corresponding docker configuration like `RemoteRuntimeEndpoint` and `RemoteImageEndpoint` in EdgeCore, then install the cri-dockerd tools as docs below: 
  https://github.com/kubeedge/kubeedge/issues/4843
  
