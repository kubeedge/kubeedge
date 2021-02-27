  * [v1.6.0](#v160)
     * [Downloads for v1.6.0](#downloads-for-v160)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
     * [KubeEdge v1.6 Release Notes](#kubeedge-v16-release-notes)
        * [1.6 What's New](#15-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


# v1.6.0

## Downloads for v1.6.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |


## KubeEdge v1.6 Release Notes

### 1.6 What's New

**Support Autonomic Kube-API Endpoint for Applications On Edge Nodes [Alpha]**

Autonomic Kube-API Endpoint is now available on edge nodes!
Users are now able to run third-party plugins and applications that depends on Kubernetes APIs on edge nodes.
List-watch connections are established between client and the local endpoint provided by EdgeCore.
With reliable message delivery and data autonomy provided by KubeEdge,
list-watch connections on edge nodes keep available even when nodes are located in high latency network or frequently get disconnected to the Cloud.

This is very useful in cases that users want to install customized versions of Kubelet, Kube-Proxy, CNI and CSI plugins with KubeEdge.
Particularly, Kubernetes CRDs are also supported on edge nodes.
([#2508](https://github.com/kubeedge/kubeedge/pull/2508), [#2587](https://github.com/kubeedge/kubeedge/pull/2587), [@GsssC](https://github.com/GsssC), [@fisherxu](https://github.com/fisherxu))

**Custom Message Routing between Cloud and Edge for Applications [Alpha]**

Added support of routing management with Rule, RuleEndpoint API and a router module.
Users are now able to use KubeEdge to deliver their custom messages between cloud and edge.

Note that it's designed for control data exchange between cloud and edge, not suitable for large data delivery.
The data size of delivery at one time is limited to 12MB.

Refer to https://kubeedge.io/en/docs/developer/custom_message_deliver/ for more details.
([#2413](https://github.com/kubeedge/kubeedge/pull/2413), [@liufen90](https://github.com/liufen90), [@WintonChan](https://github.com/WintonChan))

**Simplified Application Autonomy Configuration When Node Is Off-line**

If user wants any application to stay on edge nodes when disconnected to the cloud,
simply add label `app-offline.kubeedge.io=autonomy` to its pods.
KubeEdge will automatically override pod default toleration configuration for
Taint `node.kubernetes.io/unreachable` to avoid Kubernetes evicting pods from unreachable nodes.
([#2499](https://github.com/kubeedge/kubeedge/pull/2499), [@daixiang0](https://github.com/daixiang0))

**New home for Device Mappers code**

Device Mappers implementations now have a new home [kubeedge/mappers-go](https://github.com/kubeedge/mappers-go).

**OPC-UA Device Mapper**

OPC-UA Device Mapper with Golang implementation is provided, based on new Device Mapper Standard.
([mappers-go#4](https://github.com/kubeedge/mappers-go/pull/4), [@sailorvii](https://github.com/sailorvii)).


### Important Steps before Upgrading

NA


### Other Notable Changes

- Metamanager remote query timeout configurable ([#2336](https://github.com/kubeedge/kubeedge/pull/2336), [@lvchenggang](https://github.com/lvchenggang))
- Add unsubscribe case in eventbus ([#2345](https://github.com/kubeedge/kubeedge/pull/2345), [@muxuelan](https://github.com/muxuelan))
- upgrade klog@0.4.0 to klog/v2@2.2.0 ([#2349](https://github.com/kubeedge/kubeedge/pull/2125), [@GsssC](https://github.com/GsssC))
- Keadm: optimize OS detect ([#2388](https://github.com/kubeedge/kubeedge/pull/2388), [@daixiang0](https://github.com/daixiang0))
- Get EdgeNode ip before registerModules to fix stream module problem ([#2389](https://github.com/kubeedge/kubeedge/pull/2389), [@lvchenggang](https://github.com/lvchenggang))
- keadm: support init kubeedge with package manager `pacman` ([#2415](https://github.com/kubeedge/kubeedge/pull/2415), [@gccio](https://github.com/gccio))
- support kubectl get --raw /api/v1/nodes/{node}/proxy/metrics ([#2437](https://github.com/kubeedge/kubeedge/pull/2437), [@Abirdcfly](https://github.com/Abirdcfly))
- add func that make subscribed topics persistence ([#2457](https://github.com/kubeedge/kubeedge/pull/2457), [@muxuelan](https://github.com/muxuelan))
- edgecore: support customize node labels, taints and annotations ([#2463](https://github.com/kubeedge/kubeedge/pull/2463), [@gccio](https://github.com/gccio))
- support more metric path in cloud ([#2482](https://github.com/kubeedge/kubeedge/pull/2482), [@Abirdcfly](https://github.com/Abirdcfly))
- edgecore: add nfs localpath support ([#2529](https://github.com/kubeedge/kubeedge/pull/2529), [@swartz-k](https://github.com/swartz-k))


### Bug Fixes

- Fix a bug of device update ([#2360](https://github.com/kubeedge/kubeedge/pull/2360), [@wuqihui0317](https://github.com/wuqihui0317))
- Fix resource version compare error ([#2373](https://github.com/kubeedge/kubeedge/pull/2373), [@threestoneliu](https://github.com/threestoneliu))
- Fix msg in nodestore compare error ([#2387](https://github.com/kubeedge/kubeedge/pull/2387), [@threestoneliu](https://github.com/threestoneliu))
- fix message send problem ([#2392](https://github.com/kubeedge/kubeedge/pull/2392), [@threestoneliu](https://github.com/threestoneliu))
- fix synccontroller manage same name object error ([#2393](https://github.com/kubeedge/kubeedge/pull/2393), [@threestoneliu](https://github.com/threestoneliu))
- Fix edgehub synckeeper use unbuffer channel error ([#2414](https://github.com/kubeedge/kubeedge/pull/2414), [@threestoneliu](https://github.com/threestoneliu))
- Fix bug: keadm doesn't delete file directly when checkSum is failed ([#2446](https://github.com/kubeedge/kubeedge/pull/2446), [@XiaoJiangWang](https://github.com/XiaoJiangWang))
- cloudstream: fix panic of concurrent map read and map write. ([#2454](https://github.com/kubeedge/kubeedge/pull/2454), [@gccio](https://github.com/gccio))
- Fix bug: the websocket connection timeout setting doesn't take effect ([#2471](https://github.com/kubeedge/kubeedge/pull/2471), [@XiaoJiangWang](https://github.com/XiaoJiangWang))
- cloudcore: fix panic in cloudcore ([#2552](https://github.com/kubeedge/kubeedge/pull/2552), [@gccio](https://github.com/gccio))
- fix: missing invoke StartGarbageCollection func bug ([#2563](https://github.com/kubeedge/kubeedge/pull/2563), [@hackers365](https://github.com/hackers365))
- remove cache when configmap not found from k8s ([#2582](https://github.com/kubeedge/kubeedge/pull/2582), [@guanzydev](https://github.com/guanzydev))
- Fix kubelet accessing through edge-side meta server ([#2587](https://github.com/kubeedge/kubeedge/pull/2587), [@fisherxu](https://github.com/fisherxu))

