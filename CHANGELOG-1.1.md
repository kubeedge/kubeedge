
  * [v1.1.0](#v110)
     * [Downloads for v1.1.0](#downloads-for-v110)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
        * [EdgeSite Binaries](#edgesite-binaries)
     * [KubeEdge v1.1 Release Notes](#kubeedge-v11-release-notes)
        * [1.1 What's New](#11-whats-new)
        * [Known Issues](#known-issues)
        * [Other notable changes](#other-notable-changes)

# v1.1.0

## Downloads for v1.1.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.1.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.1.0/kubeedge-v1.1.0-linux-amd64.tar.gz) | 76.3 MB | `f28aebb6ac26e859f8f2510987dd727e018a3c6cd3bdbb66464228cd2f32387cc4d216275cd3cba9c78216f143a611ad3b3c7e2bfc70648da6f73f46d6f85084` |
| [kubeedge-v1.1.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.1.0/kubeedge-v1.1.0-linux-arm.tar.gz) | 70.9 MB | `4fafd8447dca3466a42d145f3e229f186d8725b53ec6a51681b9f79cdda4b955bef7494c7b7a68513eccb9393383a50c850b76e3b678ae1da1b198b891d9e09e` |

### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.1.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.1.0/keadm-v1.1.0-linux-amd64.tar.gz) | 8.27 MB | `0f17124675adcb2396771b5ac872cf60b09b32e5d18ab5909c243c7a97adb56acd428e21df3073ec5df701ce08ae8dead98f7cd4b38e323cc9aed60c97c0df41` |

### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [edgesite-v1.1.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.1.0/edgesite-v1.1.0-linux-amd64.tar.gz) | 27.4 MB | `a16d7f5e2f45ecb0f500f794c4f07ea74b2663bcac8008711a44b3cdea34d1f104f38c1c147ca90f3ab1ff7cf529d4445c1da34c3eff3386fc9b98f45ee173be` |
| [edgesite-v1.1.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.1.0/edgesite-v1.1.0-linux-arm.tar.gz) | 25.1 MB | `facef575c67759d234568fef16cf7d8e2797874520f845817b56372a45413448aa55a9014d33de5535dd7a73b4d521ad5ebdedaff9f147e838a7f8d2887f4bf4` |

## KubeEdge v1.1 Release Notes

### 1.1 What's New

**Container Storage Interface (CSI) support**

This feature enables running applications with persistent data store at edge and KubeEdge to support [basic CSI Volume Lifecycle](https://github.com/container-storage-interface/spec/blob/master/spec.md#volume-lifecycle) and is compatible with Kubernetes and CSI.

**Dynamic Admission Control Webhook**

Admission control webhook is an effective way of validating/mutating the object configuration for KubeEdge API objects like devicemodels, devices.

**Kubernetes Upgrade**

Upgrade the venderod kubernetes version to v1.15.3, so users can use the feature of new version on the cloud and on the edge side.

**KubeEdge local setup scripts**

A bash script that can start a KubeEdge cluster in a VM with cloudcore, edgecore binaries and kind. It uses kind to start K8s cluster and runs cloudcore, edgecore binaries as processes in a single VM.

### Known Issues

- Reliable message delivery between cloud and edge is missing.

- There is no logic to partition the configmap containing multiple device models and device instances.

### Other notable changes

- Add New Feature: support dockershim in edged. ([#829](https://github.com/kubeedge/kubeedge/pull/829), [@arcanique](https://github.com/arcanique))

- Fix edge_core cannot connect to edgecontroller after disconnecting once ([#870](https://github.com/kubeedge/kubeedge/pull/870), [@shouhong](https://github.com/shouhong))

- Raspberry Pi3/4 cross build ([#903](https://github.com/kubeedge/kubeedge/pull/903), [@subpathdev](https://github.com/subpathdev))

- Upgrade to Kubernetes v1.15 ([#941](https://github.com/kubeedge/kubeedge/pull/941), [@edisonxiang](https://github.com/edisonxiang))

- Use go mod ([#947](https://github.com/kubeedge/kubeedge/pull/947), [@subpathdev](https://github.com/subpathdev))

- Rename device dir to mappers which places mappers ([#966](https://github.com/kubeedge/kubeedge/pull/966), [@fisherxu](https://github.com/fisherxu))

- New feature: L4 Proxy support in edgemesh ([#970](https://github.com/kubeedge/kubeedge/pull/970), [@arcanique](https://github.com/arcanique))

- Initialize feature lifecycle doc ([#850](https://github.com/kubeedge/kubeedge/pull/850), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))

- Change go version from go 1.11 to go 1.12 ([#982](https://github.com/kubeedge/kubeedge/pull/982), [@kadisi](https://github.com/kadisi))

- Add admission webhook for validate device CRD ([#984](https://github.com/kubeedge/kubeedge/pull/984), [@chendave](https://github.com/chendave))

- Rename edgecontroller to cloudcore  ([#988](https://github.com/kubeedge/kubeedge/pull/988), [@fisherxu](https://github.com/fisherxu))

- Rename edge_core to edgecore ([#999](https://github.com/kubeedge/kubeedge/pull/999), [@kexun](https://github.com/kexun))

- Unifying logging library to klog  ([#1019](https://github.com/kubeedge/kubeedge/pull/1019), [@kadisi](https://github.com/kadisi))

- Add in-tree csi plugin implementations ([#1047](https://github.com/kubeedge/kubeedge/pull/1047), [@edisonxiang](https://github.com/edisonxiang))

- Add csi driver from kubeedge ([#1059](https://github.com/kubeedge/kubeedge/pull/1019), [@edisonxiang](https://github.com/edisonxiang))

- Add local up kubeedge script ([#1085](https://github.com/kubeedge/kubeedge/pull/1085), [@fisherxu](https://github.com/fisherxu))

- Fix panic: concurrent write to websocket ([#1112](https://github.com/kubeedge/kubeedge/pull/1112), [@fisherxu](https://github.com/fisherxu))
