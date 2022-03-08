
* [v1.10.0](#v1100)
    * [Downloads for v1.10.0](#downloads-for-v1100)
    * [KubeEdge v1.10 Release Notes](#kubeedge-v110-release-notes)
        * [1.10 What's New](#110-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


    

# v1.10.0

## Downloads for v1.10.0

Download v1.10.0 in the [v1.10.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.10.0).

## KubeEdge v1.10 Release Notes

## 1.10 What's New

### Installation Experience Improvement with Keadm

Keadm adds some new sub-commands to improve the user experience, including containerized deployment, offline installation, etc. New sub-commands including: beta, config.

`beta` provides some sub-commands that are still in testing, but have complete functions and can be used in advance. Sub-commands including: beta init, beta manifest generate, beta join, beta reset.

- `beta init`: CloudCore Helm Chart is integrated in beta init, which can be used to deploy containerized CloudCore.

- `beta join`: Installing edgecore as system service from docker image, no need to download from github release.

- `beta reset`: Reset the node, clean up the resources installed on the node by `beta init` or `beta join`. It will automatically detect the type of node to clean up.

- `beta manifest generate`: Generate all the manifests to deploy the cloudside components.



`config` is used to configure kubeedge cluster, like cluster upgrade, API conversion, image preloading. 
Now the image preloading has supported, sub-commands including: config images list, config images pull.

- `config images list`: List all images required for kubeedge installation.

- `config images pull`: Pull all images required for kubeedge installation.

Refer to the links for more details.
([#3517](https://github.com/kubeedge/kubeedge/issues/3517), [#3540](https://github.com/kubeedge/kubeedge/pull/3540),
[#3554](https://github.com/kubeedge/kubeedge/pull/3554), [#3534](https://github.com/kubeedge/kubeedge/pull/3534))


### Preview version for Next-gen Edged: Suitable for more scenarios

A new version of the lightweight engine Edged, which is optimized from kubelet and integrated in edgecore, and occupies less resource.
Users can customize lightweight optimization according to their needs.

Refer to the links for more details.
(Dev-Branch for previewing: [feature-new-edged](https://github.com/kubeedge/kubeedge/tree/feature-new-edged))


### Edgemark: Support large-scale KubeEdge cluster performance testing

Edgemark is a performance testing tool inherited from Kubemark. The primary use case of Edgemark is also scalability testing, 
it allows users to simulate edge clusters, which can be much bigger than the real ones.

Edgemark consists of two parts: real cloud part components and a set of "Hollow" Edge Nodes. In "Hollow" Edge Nodes, EdgeCore runs in container.
The edged module runs with an injected mock CRI part that doesn't do anything. 
So the hollow edge node doesn't actually start any containers, and also doesn't mount any volumes. 


Refer to the links for more details.
([#3637](https://github.com/kubeedge/kubeedge/pull/3637))


### EdgeMesh proxy tunnel supports quic

Users can choose edgemesh's proxy tunnel as quic protocol to transmit data. In edge scenarios, nodes are often in a weak network environment.
Compared with the traditional tcp protocol, the quic protocol has better performance and QoS in the weak network environment.

Refer to the links for more details.
([#281](https://github.com/kubeedge/edgemesh/pull/281))


### EdgeMesh supports proxy for udp applications

Some users' services use the udp protocol, and now edgemesh can also support the proxy of udp applications.

Refer to the links for more details.
([#295](https://github.com/kubeedge/edgemesh/pull/295))


### EdgeMesh support SSH login between cloud-edge/edge-edge nodes

Edge nodes are generally distributed in the Private network environment, but it is often necessary to ssh login and operate the edge node.
EdgeMesh provide a socks5proxy based on the tunnel inside EdgeMesh, which supports forwarding ssh requests from cloud/edge nodes to edge nodes.

Refer to the links for more details.
([#258](https://github.com/kubeedge/edgemesh/pull/258), [#242](https://github.com/kubeedge/edgemesh/pull/242))


### Kubernetes Dependencies Upgrade

Upgrade the vendered kubernetes version to v1.22.6, users now can use the feature of new version
on the cloud and on the edge side.

Refer to the links for more details.
([#3624](https://github.com/kubeedge/kubeedge/pull/3624))



## Important Steps before Upgrading

If you want to deploy the KubeEdge v1.10.0, please note that the Kubernetes dependency is 1.22.6.


## Other Notable Changes

- Remove dependency on os/exec and curl in favor of net/http ([#3409](https://github.com/kubeedge/kubeedge/pull/3409), [@mjlshen](https://github.com/mjlshen))
- Optimize script when create stream cert ([#3412](https://github.com/kubeedge/kubeedge/pull/3412), [@gujun4990](https://github.com/gujun4990))
- Cloudhub: prevent dropping volume messages ([#3457](https://github.com/kubeedge/kubeedge/pull/3457), [@moolen](https://github.com/moolen))
- Modify the log view command after edgecore is running ([#3456](https://github.com/kubeedge/kubeedge/pull/3456), [@zc2638](https://github.com/zc2638))
- Optimize the iptables manager ([#3461](https://github.com/kubeedge/kubeedge/pull/3461), [@zhu733756](https://github.com/zhu733756))
- Add script for build release ([#3467](https://github.com/kubeedge/kubeedge/pull/3467), [@gy95](https://github.com/gy95))
- Using lateset codes to do keadm_e2e ([#3469](https://github.com/kubeedge/kubeedge/pull/3469), [@gy95](https://github.com/gy95))
- Change the resourceType of msg issued by synccontroller ([#3496](https://github.com/kubeedge/kubeedge/pull/3496), [@Rachel-Shao](https://github.com/Rachel-Shao))
- Add a basic image for building various components of KubeEdge ([#3513](https://github.com/kubeedge/kubeedge/pull/3513), [@zc2638](https://github.com/zc2638))
- Supporting crossbuild all components ([#3515](https://github.com/kubeedge/kubeedge/pull/3515), [@fisherxu](https://github.com/fisherxu))
- support multi architecture image build ([#3530](https://github.com/kubeedge/kubeedge/pull/3530), [@gy95](https://github.com/gy95))
- filter pod by nodeName ([#3594](https://github.com/kubeedge/kubeedge/pull/3594), [@yz271544](https://github.com/yz271544))


## Bug Fixes

- Fix keadm check mqtt install bug ([#3359](https://github.com/kubeedge/kubeedge/pull/3359), [@jidalong](https://github.com/jidalong))
- beehive: fixes SendSync channel allocation behaviour ([#3413](https://github.com/kubeedge/kubeedge/pull/3413), [@ankitrgadiya](https://github.com/ankitrgadiya))
- Fix deviceTwin device update fields ([#3415](https://github.com/kubeedge/kubeedge/pull/3415), [@TianTianBigWang](https://github.com/TianTianBigWang))
- Fix pod exec is not close conn when edgecore is closed ([#3417](https://github.com/kubeedge/kubeedge/pull/3417), [@lvfei103650](https://github.com/lvfei103650))
- Fixes imagePullSecrets in cloudcore deployment and in iptablesManager daemonset ([#3443](https://github.com/kubeedge/kubeedge/pull/3443), [@zivAnyvision](https://github.com/zivAnyvision))
- fix crd create failed  ([#3444](https://github.com/kubeedge/kubeedge/pull/3444), [@gy95](https://github.com/gy95))
- Identify session not by IP of node for exec ([#3499](https://github.com/kubeedge/kubeedge/pull/3499), [@stingshen](https://github.com/stingshen))
- Fix the issue when cross-building for ARMv8 ([#3522](https://github.com/kubeedge/kubeedge/pull/3522), [@chendave](https://github.com/chendave))
- fix lots of duplicate logs "Failed to get obj" in objectsync.go ([#3570](https://github.com/kubeedge/kubeedge/pull/3570), [@vincentgoat](https://github.com/vincentgoat))
- fix remove os.Exit in updateNodeKubeletEndpoint ([#3571](https://github.com/kubeedge/kubeedge/pull/3571), [@lvfei103650](https://github.com/lvfei103650))
- fix: add missing metrics func ([#3581](https://github.com/kubeedge/kubeedge/pull/3581), [@moolen](https://github.com/moolen))
- Fix pv's objectsync spec.ObjectKind is empty ([#3599](https://github.com/kubeedge/kubeedge/pull/3599), [@neiba](https://github.com/neiba))
- fix memory leak in edge kube-api interface ([#3605](https://github.com/kubeedge/kubeedge/pull/3605), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- pv's objectsync spec.ObjectKind is empty ([#3613](https://github.com/kubeedge/kubeedge/pull/3613), [@neiba](https://github.com/neiba))
