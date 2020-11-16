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


### Installer Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |


### EdgeSite Binaries
| filename | Size | sha512 hash |
| -------- | ---- | ----------- |


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
