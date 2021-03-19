  * [v1.5.0](#v150)
     * [Downloads for v1.5.0](#downloads-for-v150)
        * [KubeEdge Binaries](#kubeedge-binaries)
        * [Installer Binaries](#installer-binaries)
        * [EdgeSite Binaries](#edgesite-binaries)
     * [KubeEdge v1.5 Release Notes](#kubeedge-v15-release-notes)
        * [1.5 What's New](#15-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)


# v1.5.0

## Downloads for v1.5.0

### KubeEdge Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [kubeedge-v1.5.0-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/kubeedge-v1.5.0-linux-arm64.tar.gz) |  79.3 MB | 2dc2f4e6a0d79a68321c8a9159e063223a1822bcc56084e99222d5660a2187356b11d0124d05d12ad2b6e536d013d0e016f07d12d2ef42ab703e1fba18ce8805 |
| [kubeedge-v1.5.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/kubeedge-v1.5.0-linux-arm.tar.gz) | 78 MB | 8392850dc4b186ac90720f463977ae55b356cfbf889125d86f344283e514f65414d93505c1f21a5f9dc9633cc0e6bdb16fe68cc6b8d8055c880d82b641c97db8 |
| [kubeedge-v1.5.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/kubeedge-v1.5.0-linux-amd64.tar.gz) | 45.7 MB | 3b006f32bfbccc7f4c12759dd560faa37655bdcb16bcb590574d9f5ed2fe6dd06e5e1a5dd8e7bb3fdf248794eacd8b13e7c8a3edfadb17597f7bf35620cf3159 |


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [keadm-v1.5.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/keadm-v1.5.0-linux-arm64.tar.gz) |  20.1 MB | 24b80603891c8a6f23ef593fde084d2b6629e1c0d7a33d8770ba4002d7083b6c4c8bfce6a1975ca1a94ff19402a8dbbcd5954cd399312e3fb2958afcf9878461 |
| [keadm-v1.5.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/keadm-v1.5.0-linux-arm.tar.gz) |  19.8 MB | 0b76b72bd1fb74b57d9b8b4edc350f59e7935ad2a07cf225fc5103a5552fbc1b0ee0cd8db7e5afd1c0eab06af4dcc7a0879ea5e3b16073acec0efeecfb429bec |
| [keadm-v1.5.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/keadm-v1.5.0-linux-amd64.tar.gz) |  11.7 MB | 650122a02f0973785c0e65988bdc493be17597e8fb159028edbcdbc0355801772a90bdf4b822f8fb905f626958c0549137e2f39d07ccca0a2f611b7d7941e627 |


### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |
| [edgesite-v1.5.0-linux-arm64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/edgesite-v1.5.0-linux-arm64.tar.gz) | 28.1 MB | 95f6e946dc7172ecd94a2607a7bdc8d39046e513c975c8bd3c88652c5800df3465fb620e5ac2558b22c861ec95bdad7842333d4d95e53e5ab27bcf96039ae054 |
| [edgesite-v1.5.0-linux-arm.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/edgesite-v1.5.0-linux-arm.tar.gz) | 27.7 MB | 94043bc337a5dda70811dc6fd1e7800e418abf31ec6a442980182c5767ab86886acbbc12e34951f580be0bcd750e3b35ac69211af48379ea9325a18c25326654 |
| [edgesite-v1.5.0-linux-amd64.tar.gz](https://github.com/kubeedge/kubeedge/releases/download/v1.5.0/edgesite-v1.5.0-linux-amd64.tar.gz) | 16.5 MB | e9eff446996c1df7020d161cf0945ed4bd378f23fadf5e3c77c84c3b0b7d6255fbb55482dcb23a6bf2e74ff3f953b901c42cf28e5a6fb9adf3e6e4651c9e6832 |


## KubeEdge v1.5 Release Notes

### 1.5 What's New

**Simplified Device Mapper reference architecture**

New version of Mapper reference architecture:

- Simplified Mapper code structure
- Extracted common code into SDK
- Added new building blocks: Configmap parser, Driver, Event process, Timer

Users are now able to develop mappers based on the new design standard.([#2147](https://github.com/kubeedge/kubeedge/pull/2147), [@sailorvii](https://github.com/sailorvii), [@luogangyi](https://github.com/luogangyi)).

**Modbus Mapper Golang Implementation**

A new modbus mapper with Golang implementation is provided, based on new Device Mapper Standard. ([#2282](https://github.com/kubeedge/kubeedge/pull/2282), [@sailorvii](https://github.com/sailorvii)). 

**Support Remote Exec to Pods on Edge From Cloud**

Users are now able to use `K8s exec api` or `kubectl exec` command to connect to pods on the edge node. ([#2075](https://github.com/kubeedge/kubeedge/pull/2075), [@daixiang0](https://github.com/daixiang0), [@kadisi](https://github.com/kadisi)).

**Support Keadm Debug Command for Trouble Shooting On Edge Nodes**

A set of keadm debug subcommands are added for Trouble Shooting On Edge Nodes.
Users are now able to use `keadm debug get` and `keadm debug collect` to get/collect KubeEdge local data for trouble shooting, 
and use `keadm debug check` and `keadm debug diagnose` to check local environment configuration. ([#1939](https://github.com/kubeedge/kubeedge/pull/1939), [@shenkonghui](https://github.com/shenkonghui), [@qingchen1203](https://github.com/qingchen1203))

**Kubernetes Dependencies Upgrade**

Upgrade the vendered kubernetes version to v1.19.3, users now can use the feature of new version
on the cloud and on the edge side. ([#2223](https://github.com/kubeedge/kubeedge/pull/2223), [@dingyin](https://github.com/dingyin), [@zzxgzgz](https://github.com/zzxgzgz))

### Important Steps before Upgrading

NA

### Other Notable Changes

- Eventbus add tls config when connect to mqtt ([#2109](https://github.com/kubeedge/kubeedge/pull/2109), [@lvchenggang](https://github.com/lvchenggang))
- Add zero judgment when pods is obtained from the cache ([#2115](https://github.com/kubeedge/kubeedge/pull/2115), [@XiaoJiangWang](https://github.com/XiaoJiangWang))
- Add metrics api support in streamserver ([#2125](https://github.com/kubeedge/kubeedge/pull/2125), [@luogangyi](https://github.com/luogangyi))
- Support config domain URL for cloudcore ([#2126](https://github.com/kubeedge/kubeedge/pull/2126), [@ls889](https://github.com/ls889))
- Keadm: support arm/arm64 for CentOS ([#2149](https://github.com/kubeedge/kubeedge/pull/2149), [@daixiang0](https://github.com/daixiang0))
- Cloudcore readiness gate show error status ([#2157](https://github.com/kubeedge/kubeedge/pull/2157), [@Yellow-L](https://github.com/Yellow-L))
- Added the function of unsubscribe topics ([#2188](https://github.com/kubeedge/kubeedge/pull/2188), [@muxuelan](https://github.com/muxuelan))
- Keadm: add command line option crgoupdriver for subcommand join ([#2202](https://github.com/kubeedge/kubeedge/pull/2202), [@Yellow-L](https://github.com/Yellow-L))
- Add restart options for cloudcore.service and edgecore.service ([#2207](https://github.com/kubeedge/kubeedge/pull/2207), [@YaozhongZhang](https://github.com/YaozhongZhang))
- Add a option "--package-path" to keadm init/join ([#2213](https://github.com/kubeedge/kubeedge/pull/2213), [@Rachel-Shao](https://github.com/Rachel-Shao))
- Keadm: add debian OS support ([#2234](https://github.com/kubeedge/kubeedge/pull/2234), [@daixiang0](https://github.com/daixiang0))



### Bug Fixes

- Edged support update pod status after consume added pod ([#2108](https://github.com/kubeedge/kubeedge/pull/2108), [@lvchenggang](https://github.com/lvchenggang))
- Fix resource version compare ([#2120](https://github.com/kubeedge/kubeedge/pull/2120), [@luogangyi](https://github.com/luogangyi))
- Fix/keadm: checksum validation of the downloaded file every time ([#2135](https://github.com/kubeedge/kubeedge/pull/2135), [@ttlv](https://github.com/ttlv))
- Fix deviceProfile json bug ([#2143](https://github.com/kubeedge/kubeedge/pull/2143), [@jidalong](https://github.com/jidalong))
- Fix reDownload serviceFile ([#2170](https://github.com/kubeedge/kubeedge/pull/2170), [@GsssC](https://github.com/GsssC))
- Fix bug: cloud send twin_update confirm message to edge ([#2182](https://github.com/kubeedge/kubeedge/pull/2182), [@jidalong](https://github.com/jidalong))
- Fix edged log issue ([#2227](https://github.com/kubeedge/kubeedge/pull/2227), [@daixiang0](https://github.com/daixiang0))
- Fix edgecore service download issue under proxy environment ([#2253](https://github.com/kubeedge/kubeedge/pull/2253), [@llhuii](https://github.com/llhuii))
- Fix missing protocol common config update ([#2265](https://github.com/kubeedge/kubeedge/pull/2265), [@luogangyi](https://github.com/luogangyi))
- Fix a bug of device data type ([#2285](https://github.com/kubeedge/kubeedge/pull/2285), [@wuqihui0317](https://github.com/wuqihui0317))
- Fix cloudhub api /readyz panic ([#2304](https://github.com/kubeedge/kubeedge/pull/2304), [@gccio](https://github.com/gccio))
- Bugfix:less verbose edge message output ([#2318](https://github.com/kubeedge/kubeedge/pull/2318), [@tangyanhan](https://github.com/tangyanhan))
