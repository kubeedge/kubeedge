  * [v1.3.1](#v131)
     * [Downloads for v1.3.1](#downloads-for-v131)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
        * [EdgeSite Binaries](#edgesite-binaries)
     * [KubeEdge v1.3.1 Release Notes](#kubeedge-v131-release-notes)
        * [Changelog since v1.3.0](#changelog-since-v130)
  * [v1.3.0](#v130)
     * [Downloads for v1.3.0](#downloads-for-v130)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
        * [EdgeSite Binaries](#edgesite-binaries)
     * [KubeEdge v1.3 Release Notes](#kubeedge-v13-release-notes)
        * [1.3 What's New](#13-whats-new)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)
  * [v1.3.0-beta.0](#v130-beta0)
     * [Changelog since v1.2.0](#changelog-since-v120)
        * [Bug Fixes](#bug-fixes-1)

# v1.3.1

## Downloads for v1.3.1

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.3.1-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/kubeedge-v1.3.1-linux-arm64.tar.gz) |  76 MB | b1392dc2813078482dbb769ba21dd4c5583eb55f3ca3bfb531923ad4a4f080f12f411afe637840946198ba8b93345f2355d0f29414c4aea456f67265f467bdf2 |
| [kubeedge-v1.3.1-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/kubeedge-v1.3.1-linux-arm.tar.gz) | 74.9 MB | 2e7312d16583664d4ddb125630ea2612eabc1df4e935ff8c1d4cf1b92e8a3968dde8b2123f6b4f8d1e029caa385f2b0dfd7f92c133d84c9953e5dcea1b479a40 |
| [kubeedge-v1.3.1-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/edgesite-v1.3.1-linux-amd64.tar.gz) | 80.7 MB | 00c03e5343399ce518ed0445f5d400cfe908429c26305768b1e184787b55364aa256b44f335fa4a124f11bf6fd18c4a9d7a8f7b9fec932650d12469bb743b2a9 |


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.3.1-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/keadm-v1.3.1-linux-amd64.tar.gz) |  15.1 MB | 6f2d55f4fcb21da6b71ef5ed371e82deb9cb04fa079dee3692861cbbe3ec00eeebd237e26631218085c01ab66f4332d4a47c4cfe4080f115e8396a2dcf74e3b1 |

### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [edgesite-v1.3.1-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/edgesite-v1.3.1-linux-arm64.tar.gz) | 27.5 MB | 06a221a5bdfacc3a5dbbae75d5518b5e3fcbf7ca27d6623021747f53136a33d6994dc6281b0b2b5752e5cd53d71434efeb5cd98e0fe31b1d8b65fb7fedbd3bd0 |
| [edgesite-v1.3.1-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/edgesite-v1.3.1-linux-arm.tar.gz) | 27.2 MB | 39350b8bfd0b0f434296b9634e4727a3dd54fc8d45b453ad2479aaeb259ece634dc4049b00ff93a47bff674fd0c34f9ef99caab6d8ccb11b34e391df1409d801 |
| [edgesite-v1.3.1-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.1/edgesite-v1.3.1-linux-amd64.tar.gz) | 29.8 MB | b45bccbd2d0f76991325838cdba84c21eebed85c6d33b893b92d468a052e424b72a1e81a452e9e4e13cae25808c7ccb174b68102e647903da1ed609a1fb3e96f |


## KubeEdge v1.3.1 Release Notes

### Changelog since v1.3.0

- Expose readyz checker in cloudcore. ([#1663](https://github.com/kubeedge/kubeedge/pull/1663), [@Poor12](https://github.com/Poor12))

- Configure tls use ECDSA cipher and fix handshake failure. ([#1702](https://github.com/kubeedge/kubeedge/pull/1702), [@luogangyi](https://github.com/luogangyi))

- Automatically create kubeedge namespace when creating a secret. ([#1705](https://github.com/kubeedge/kubeedge/pull/1705), [@ls889](https://github.com/ls889))

- Fix using same namespace while syncPodStatus. ([#1708](https://github.com/kubeedge/kubeedge/pull/1708), [@GsssC](https://github.com/GsssC))

- Fix edgemesh caching serviceList bug. ([#1757](https://github.com/kubeedge/kubeedge/pull/1757), [@liuzhiyi1993](https://github.com/liuzhiyi1993))

- Fix certs issue for kubectl logs after V1.3. ([#1697](https://github.com/kubeedge/kubeedge/pull/1697), [@ls889](https://github.com/ls889))

- Update keadm to support KubeEdge 1.3 and pre 1.3 releases. ([#1701](https://github.com/kubeedge/kubeedge/pull/1701), [@XJangel](https://github.com/XJangel))

# v1.3.0

## Downloads for v1.3.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.3.0-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/untagged-1c53f41b984950b3ac04/kubeedge-v1.3.0-linux-arm64.tar.gz) |  82.1 MB | bdbb17fcde5d8f08c686be76e90c874a34aa06d27e7aca49e6d965edd8d2d3a07cfd482791ab22d248d5eac78d1cc3b097b94ca44c0f45e83784b73f2f23df81 |
| [kubeedge-v1.3.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/untagged-1c53f41b984950b3ac04/kubeedge-v1.3.0-linux-arm.tar.gz) | 83.7 MB | 70808681d50e08a0e84fa84b8355234c75865bd2c057f27afd806fd0d2b6b068108ff7272be5ce927a81d176bfa5e89576aea0db08022d7414a5d77e25da83a6 |
| [kubeedge-v1.3.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/untagged-1c53f41b984950b3ac04/kubeedge-v1.3.0-linux-amd64.tar.gz) | 81.00 MB | d3ef10a39f8cf9c15f8121d9ae9e0b416c1d72ff24689ece17cdacb60fea6a8d06633a4779f4b1185ffe7ee15e230a98bb2da0d46b088a7b45c339a41e5bab02 |


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.3.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.3.0/keadm-v1.3.0-linux-amd64.tar.gz) |  15.1 MB | ec86e027abf5602b90a90557cc3153dcd85fb5350ba0e57a6bd607606d7a0e1165535548b11c31df70db52f2a9aef3604f6be6b8e3d097afeeda9d845bdee507 |

### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [edgesite-v1.3.0-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/untagged-1c53f41b984950b3ac04/edgesite-v1.3.0-linux-arm64.tar.gz) | 29.54 MB | ccf2d846b0157202327241bf9820284583a828fb9e1a5a0f775f352f1072473ec8a40833d8f223f27d2417c54b76f4dc4e499c242830613977ccd6400519f7a1 |
| [edgesite-v1.3.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/untagged-1c53f41b984950b3ac04/edgesite-v1.3.0-linux-arm.tar.gz) | 29.24 MB | 78eb8e628d311891dbf8e1a15ef9580e32f6e2650dbb59d5ccff56e1668707ecc0fceb27f0850b668378c99da679961488a300f83cad2b12b5053acfe056aba1 |
| [edgesite-v1.3.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/untagged-1c53f41b984950b3ac04/edgesite-v1.3.0-linux-amd64.tar.gz) | 29.77 MB | 6464abf457a71f929580deb7ff1d292a3e1e79e277122256e906878e801e99b7e95c1c8be670eb35d524c5a28d54c86e1c3ef2a19584d060ffa8f4f04ccdc0b0 |



## KubeEdge v1.3 Release Notes

### 1.3 What's New

**CloudCore HA**

CloudCore now supports high availability deployment with active-standby mode ([#1600](https://github.com/kubeedge/kubeedge/pull/1600), [@GsssC](https://github.com/GsssC), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng)).

**EdgeNode auto TLS Bootstrapping**

KubeEdge is now able to generate certificates and enforce TLS between CloudCore and EdgeCore automatically. ([#1605](https://github.com/kubeedge/kubeedge/pull/1605), [@fisherxu](https://github.com/fisherxu), [@ls889](https://github.com/ls889),  [@XJangel](https://github.com/XJangel))

**Edge Pod Logs**

Users are now able to use `kubectl logs` to fetch logs from pods at edge. Follow the [instructions here](https://github.com/kubeedge/kubeedge/blob/release-1.3/docs/setup/kubeedge_install_source.md) to enable the feature. ([#1606](https://github.com/kubeedge/kubeedge/pull/1606), [@kadisi](https://github.com/kadisi)).

**Metrics at Edge**

Add metrics interfaces at edge ([#1573](https://github.com/kubeedge/kubeedge/pull/1573), [@fisherxu](https://github.com/fisherxu))

**Lightweight Runtime Support**

KubeEdge now support CRI-O as container runtime, to use less memory on edge node ([#1610](https://github.com/kubeedge/kubeedge/pull/1610), [@chendave](https://github.com/chendave))


### Other Notable Changes

- Add edge-node certs bootstrap (https://github.com/kubeedge/kubeedge/pull/1605, @fisherxu, @ls889, @XJangel)
- support kubectl logs to fetch logs of pods at edge (https://github.com/kubeedge/kubeedge/pull/1606, @kadisi)
- CloudCore now supports HA with active-standby mode (https://github.com/kubeedge/kubeedge/pull/1600, @GsssC, @kevin-wangzefeng)
- Add feature to provision VM workload (https://github.com/kubeedge/kubeedge/pull/1618, @dingyin)
- A set of component config API fields have been updated with capitalization (https://github.com/kubeedge/kubeedge/pull/1616, @kevin-wangzefeng)
- Add support for CRI-O as light-weight runtime (https://github.com/kubeedge/kubeedge/pull/1610, @chendave)
- Edge nodes are now registered with kubernetes.io/arch kubernetes.io/os labels automatically (https://github.com/kubeedge/kubeedge/pull/1601, @kevin-wangzefeng)
- Add metrics interfaces at edge (https://github.com/kubeedge/kubeedge/pull/1573, @fisherxu)
- keadm now support installing KubeEdge on CentOS (https://github.com/kubeedge/kubeedge/pull/1536, @FengyunPan2)
- edgecore: support configmap environment variable (https://github.com/kubeedge/kubeedge/pull/1518, @xmwilldo)
- EdgeMesh now doesn't rely on initContainers for initialization (https://github.com/kubeedge/kubeedge/pull/1380, liuzhiyi1993)


### Bug Fixes
- keadm: fix get version issue. ([#1628](https://github.com/kubeedge/kubeedge/pull/1628), [@daixiang0](https://github.com/daixiang0))
- Change the token separator space to dot. ([#1640](https://github.com/kubeedge/kubeedge/pull/1640), [@XJangel](https://github.com/XJangel))
- Add advitise address in cloudcore. ([#1643](https://github.com/kubeedge/kubeedge/pull/1643), [@fisherxu](https://github.com/fisherxu))
- Log errors in function NewCloudCoreCertDERandKey. ([#1645](https://github.com/kubeedge/kubeedge/pull/1645), [@ls889](https://github.com/ls889))
- Add env status.hostIP for pods. ([#1655](https://github.com/kubeedge/kubeedge/pull/1655), [@qingchen1203](https://github.com/qingchen1203))
- Fix GroupName for devices apis register ([#1594](https://github.com/kubeedge/kubeedge/pull/1594), [@bretagne-peiqi](https://github.com/bretagne-peiqi))
- Fix message handleserver close channel panic ([#1557](https://github.com/kubeedge/kubeedge/pull/1557), [@drcwr](https://github.com/drcwr))
- Fix panic in cloudcore ([#1552](https://github.com/kubeedge/kubeedge/pull/1552), [@fisherxu](https://github.com/fisherxu))



# v1.3.0-beta.0

## Changelog since v1.2.0

- Add edge-node certs bootstrap (https://github.com/kubeedge/kubeedge/pull/1605, @fisherxu, @ls889, @XJangel)
- support kubectl logs to fetch logs of pods at edge (https://github.com/kubeedge/kubeedge/pull/1606, @kadisi)
- CloudCore now supports HA with active-standby mode (https://github.com/kubeedge/kubeedge/pull/1600, @GsssC, @kevin-wangzefeng)
- Add feature to provision VM workload (https://github.com/kubeedge/kubeedge/pull/1618, @dingyin)
- A set of component config API fields have been updated with capitalization (https://github.com/kubeedge/kubeedge/pull/1616, @kevin-wangzefeng)
- Add support for CRI-O as light-weight runtime (https://github.com/kubeedge/kubeedge/pull/1610, @chendave)
- Edge nodes are now registered with kubernetes.io/arch kubernetes.io/os labels automatically (https://github.com/kubeedge/kubeedge/pull/1601, @kevin-wangzefeng)
- Add metrics interfaces at edge (https://github.com/kubeedge/kubeedge/pull/1573, @fisherxu)
- keadm now support installing KubeEdge on CentOS (https://github.com/kubeedge/kubeedge/pull/1536, @FengyunPan2)
- edgecore: support configmap environment variable (https://github.com/kubeedge/kubeedge/pull/1518, @xmwilldo)
- EdgeMesh now doesn't rely on initContainers for intialization (https://github.com/kubeedge/kubeedge/pull/1380, liuzhiyi1993)


### Bug Fixes
- fix GroupName for devices apis register (https://github.com/kubeedge/kubeedge/pull/1594, @bretagne-peiqi)
- fix message handleserver close channel panic (https://github.com/kubeedge/kubeedge/pull/1557, @drcwr)
- Fix panic in cloudcore (https://github.com/kubeedge/kubeedge/pull/1552, @fisherxu)
