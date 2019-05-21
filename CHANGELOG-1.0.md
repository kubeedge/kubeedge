
   * [v1.0.0](#v100)
      * [Downloads for v1.0.0](#downloads-for-v100)
         * [KubeEdge Binaries](#kubeedge-binaries)
         * [Installer Binaries](#installer-binaries)
         * [EdgeSite Binaries](#edgesite-binaries)
      * [KubeEdge v1.0 Release Notes](#kubeedge-v10-release-notes)
         * [1.0 What's New](#10-whats-new)
         * [Known Issues](#known-issues)
         * [Other notable changes](#other-notable-changes)
   * [v1.0.0-beta.0](#v100-beta0)
      * [Downloads for v1.0.0-beta.0](#downloads-for-v100-beta0)
         * [KubeEdge Binaries](#kubeedge-binaries-1)
         * [Installer Binaries](#installer-binaries-1)
         * [EdgeSite Binaries](#edgesite-binaries-1)
      * [Changelog since v0.3.0](#changelog-since-v030)
         * [Features Added](#features-added)
         * [Known Issues](#known-issues-1)
         * [Other notable changes](#other-notable-changes-1)

# v1.0.0

## Downloads for v1.0.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.0.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0/kubeedge-v1.0.0-linux-amd64.tar.gz) | 46.3 MB | `43839f1e539361d8eacf6b7d2c5f8664f886c15ad5b1199253b62ddbe7f1eec48e0b407992780a51c63448228977b7956e81a208daebb8c9f2ed17ed44a2ba3a` |
| [kubeedge-v1.0.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0/kubeedge-v1.0.0-linux-arm.tar.gz) | 43.4 MB | `c6635e4c61fe88a833a1d4649ea65cd98cc4a3dc2494f4b3194e2ba84f2765a4b1c4c58cab02237c9dc772c4147baea107af283318006623e8439c72ce7b7831` |

### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.0.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0/keadm-v1.0.0-linux-amd64.tar.gz) | 7.74 MB | `ed92444996665d1a1952da66047d389465e0e5930b47d78e88d6f146c8d480bc9b14a3fbac190ab70f76d89060da7856436f52240aa9b3f83b8bb990c6f15204` |

### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [edgesite-v1.0.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0/edgesite-v1.0.0-linux-amd64.tar.gz) | 24.6 MB | `afe61db4908fa67e0aadd8ab26e35224a1de8096130c2df681780befdc317331c8a2be2a298d89e6603ad36dd19e339f7256f887d1d2fd5464b3ab4c43aae3c8` |
| [edgesite-v1.0.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0/edgesite-v1.0.0-linux-arm.tar.gz) | 22.5 MB | `1ea16aad0dc0f3517d9bddcc5de16d5b46dd881ac3cc4ff7522188dad176a517fa111c3b81a223e71f45d8e69096dbedf0703d9e2592dcec292a6231e455224d` |

# KubeEdge v1.0 Release Notes

## 1.0 What's New

**Edge Mesh**

This feature aims to support service mesh capabilities on the edge to support microservice communication cross cloud and edge. In v1.0.0 release, pod-to-pod communication on the same edge node or across edge nodes in the same subnet is supported.

**CRI support**

This feature enables edged to communicate with a CRI-compliant runtime to manage containers running on resource constrained edge nodes. Support for containerd is tested.

**Quic protocol support**

In order to enhance cloud and edge communication efficiency, communication between the edge and the cloud is now also supported via QUIC, a UDP-based protocol. CloudHub supports both Websocket and QUIC protocol access at the same time. The edgehub can choose one of the protocols to access the cloudhub.

**Modbus Mapper**

Modbus Mapper is an application that is used to connect and control devices that use Modbus(RTU/TCP) as a communication protocol. The user is required to provide the mapper with the information required to control their device through the dpl configuration file. These can be changed at runtime by updating configmap.

**Edge Site**

Edge Site enables to run a standalone Kubernetes cluster at the edge along with KubeEdge to get full control and improve the offline scheduling capability. A Kubernetes cluster is deployed at edge location including the management control plane. For the management control plane to manage a reasonable scale of edge worker nodes, the host master node needs to have sufficient resources.

### Known Issues

- Reliable message delivery is missing between cloud and edge.

- Installer currently doesn't support installation of containerd, cni-plugins.

- Port mapping is not supported in CRI.

### Other notable changes

- Pods created using CRI on deletion remain in terminating status. ([#755](https://github.com/kubeedge/kubeedge/pull/755), [@gpinaik](https://github.com/gpinaik))

- fix bug: replace value in key, value by expected ([#759](https://github.com/kubeedge/kubeedge/pull/759), [@RicardoZPHuang](https://github.com/RicardoZPHuang))

- update edge_core version to reflect the vendor k8s and kubeedge version ([#761](https://github.com/kubeedge/kubeedge/pull/761), [@sids-b](https://github.com/sids-b))

- Edgesite documentation ([#737](https://github.com/kubeedge/kubeedge/pull/737), [@naveensriram](https://github.com/naveensriram))

- Moving keadm installer readme.md to Docs folder ([#745](https://github.com/kubeedge/kubeedge/pull/745), [@srivatsav123](https://github.com/srivatsav123))

- add edgemesh end to end test guide ([#758](https://github.com/kubeedge/kubeedge/pull/758), [@anyushun](https://github.com/anyushun))

- Changes made to Installer to support CRI ([#795](https://github.com/kubeedge/kubeedge/pull/795), [@srivatsav123](https://github.com/srivatsav123))

- Added documentation for device controller ([#743](https://github.com/kubeedge/kubeedge/pull/743), [@sujithsimon22](https://github.com/sujithsimon22))

# v1.0.0-beta.0

## Downloads for v1.0.0-beta.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.0.0-beta.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0-beta.0/kubeedge-v1.0.0-beta.0-linux-amd64.tar.gz) | 46.3 MB | `3394427e02eb32e3b742878abba075d3d6fb6a90b1379d1ac7925410a270c4b54b510b93ad4839b952a5bb1627b6e019d843ce28f04f84052d75d2d1176466d9` |
| [kubeedge-v1.0.0-beta.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0-beta.0/kubeedge-v1.0.0-beta.0-linux-arm.tar.gz) | 43.4 MB | `ff8736ca168570c9c14b7c83a0cb7330418b92ecf6165acd3fc633120db66e3e4b04de3e3744e1d8f6242d96da5e20e85ae49ddcc187e819c92e7f87ab0ad22a` |

### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.0.0-beta.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0-beta.0/keadm-v1.0.0-beta.0-linux-amd64.tar.gz) | 7.74 MB | `6a8aa40e7be9eea11f884e21c6c7f2c72bc8f5763eea2a6bef3aaeeba61321ac8c1dff213d5e430c4a59163dce1e4e506c8a79bc4cdc6a86c40cfad48f6c5764` |

### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [edgesite-v1.0.0-beta.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0-beta.0/edgesite-v1.0.0-beta.0-linux-amd64.tar.gz) | 24.6 MB | `2f8487acdcad6673ba7ffb849c20d5dd4959ca79b5c8407d08710a990dec3bd0a743180b723f94294de11089c9d98acbd5e4933d0f8b077ecaeae0a35cf69d71` |
| [edgesite-v1.0.0-beta.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.0.0-beta.0/edgesite-v1.0.0-beta.0-linux-arm.tar.gz) | 22.5 MB | `e9c80fe480c065a6561afe6fd60fe3515850cb7ab98d06df8c206a7ae6a541b63b5a74fce968408afc4d43a82ac0b8d227c524f002659e9b4fd4b5972260e30c` |

## Changelog since v0.3.0

### Features Added 


**Edge Mesh**

This feature aims to support service mesh capabilities on the edge to support microservice communication cross cloud and edge. In v1.0.0 release , pod-to-pod communication on the same edge node or across
edge nodes in the same subnet is supported.

**CRI support**

This feature enables edged to talk to a CRI-compliant runtime to manage containers running on resource constrained edge nodes. Support for containerd is tested.


**Quic protocol support**

In order to enhance cloud and edge communication efficiency, communication between the edge and the cloud is now also supported via QUIC , a UDP-based protocol. CloudHub supports both Websocket and QUIC protocol access at the same time. The edgehub can choose one of the protocols to access the cloudhub.


**Modbus Mapper**

Modbus Mapper is an application that is used to connect and control devices that use Modbus(RTU/TCP) as a communication protocol.  The user is required to provide the mapper with the information required to control their device through the dpl configuration file. These can be changed at runtime by updating configmap.


**Edge Site**

Edge Site enables to run a standalone Kubernetes cluster at the edge along with KubeEdge to get full control and improve the offline scheduling capability. A Kubernetes cluster is deployed at edge location including the management control plane. For the management control plane to manage a reasonable scale of edge worker nodes, the host master node needs to have sufficient resources.


### Known Issues

- Reliable message delivery is missing between cloud and edge.


### Other notable changes

- add response logic for node status updating. ([#527](https://github.com/kubeedge/kubeedge/pull/527), [@shouhong](https://github.com/shouhong))

- Consolidate the kube config at controller. ([#554](https://github.com/kubeedge/kubeedge/pull/554), [@shouhong](https://github.com/shouhong))

- GetClient return error info. ([#610](https://github.com/kubeedge/kubeedge/pull/610), [@kadisi](https://github.com/kadisi))

- removing race condition in dtcontext. ([#653](https://github.com/kubeedge/kubeedge/pull/653), [@subpathdev](https://github.com/subpathdev))

- added edge node role label while running installer ([#644](https://github.com/kubeedge/kubeedge/pull/644), [@sids-b](https://github.com/sids-b))

- fix build under macOS ([#635](https://github.com/kubeedge/kubeedge/pull/635), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))

- docs:[Fix:#608]: Open ports to cloud component for edge nodes in the installer(keadm) readme ([#632](https://github.com/kubeedge/kubeedge/pull/635), [@vizero1](https://github.com/vizero1))

- fix message lost ([#656](https://github.com/kubeedge/kubeedge/pull/656), [@edisonxiang](https://github.com/edisonxiang))

- Add file checksum comparison while downloading KubeEdge binaries ([#595](https://github.com/kubeedge/kubeedge/pull/595), [@shouhong](https://github.com/shouhong))

- KubeEdge Installer: Adding support to establish secure connectivity between edgecontroller and K8S api-server ([#557](https://github.com/kubeedge/kubeedge/pull/557), [@kadisi](https://github.com/kadisi))

- controller check error ([#599](https://github.com/kubeedge/kubeedge/pull/599), [@subpathdev](https://github.com/subpathdev))

- Issue #631: Add validation of required fields for device model and device instance ([#669](https://github.com/kubeedge/kubeedge/pull/669), [@rohitsardesai83](https://github.com/rohitsardesai83))

- Changing AccessMode values to match API validation ([#677](https://github.com/kubeedge/kubeedge/pull/677), [@sujithsimon22](https://github.com/sujithsimon22))

- Added conversion operations for read action of action manager ([#684](https://github.com/kubeedge/kubeedge/pull/684), [@sujithsimon22](https://github.com/sujithsimon22))
