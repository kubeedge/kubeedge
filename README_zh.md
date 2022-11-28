# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3018/badge)](https://bestpractices.coreinfrastructure.org/projects/3018)

![logo](./docs/images/KubeEdge_logo.png)

[English](./README.md) | 简体中文

KubeEdge 是一个开源的系统，可将本机容器化应用编排和管理扩展到边缘端设备。它基于 Kubernetes 构建，为网络和应用程序提供核心基础架构支持，并在云端和边缘端部署应用，同步元数据。KubeEdge 还支持 **MQTT** 协议，允许开发人员编写客户逻辑，并在边缘端启用设备通信的资源约束。KubeEdge 包含云端和边缘端两部分。

使用 KubeEdge，可以很容易地将已有的复杂机器学习、图像识别、事件处理和其他高级应用程序部署到边缘端并进行使用。
随着业务逻辑在边缘端上运行，可以在本地保护和处理大量数据。
通过在边缘端处理数据，响应速度会显著提高，并且可以更好地保护数据隐私。

KubeEdge 是一个由 [Cloud Native Computing Foundation](https://cncf.io) (CNCF) 托管的孵化级项目，CNCF 对 KubeEdge 的 [孵化公告](https://www.cncf.io/blog/2020/09/16/toc-approves-kubeedge-as-incubating-project/)

注意：
1.8 以前的版本不再支持，请尝试升级到支持版本。

## 优势

- **Kubernetes 原生支持**：使用 KubeEdge 用户可以在边缘节点上编排应用、管理设备并监控应用程序/设备状态，就如同在云端操作 Kubernetes 集群一样。

- **云边可靠协作**：在不稳定的云边网络上，可以保证消息传递的可靠性，不会丢失。

- **边缘自治**：当云边之间的网络不稳定或者边缘端离线或重启时，确保边缘节点可以自主运行，同时确保边缘端的应用正常运行。

- **边缘设备管理**：通过 Kubernetes 的原生API，并由CRD来管理边缘设备。

- **极致轻量的边缘代理**：在资源有限的边缘端上运行的非常轻量级的边缘代理(EdgeCore)。


## 它如何工作

KubeEdge 由云端和边缘端部分构成：

### 架构

![架构图](docs/images/kubeedge_arch.png)

### 云上部分
- [CloudHub](https://kubeedge.io/en/docs/architecture/cloud/cloudhub): CloudHub 是一个 Web Socket 服务端，负责监听云端的变化，缓存并发送消息到 EdgeHub。
- [EdgeController](https://kubeedge.io/en/docs/architecture/cloud/edge_controller): EdgeController 是一个扩展的 Kubernetes 控制器，管理边缘节点和 Pods 的元数据确保数据能够传递到指定的边缘节点。
- [DeviceController](https://kubeedge.io/en/docs/architecture/cloud/device_controller): DeviceController 是一个扩展的 Kubernetes 控制器，管理边缘设备，确保设备信息、设备状态的云边同步。


### 边缘部分
- [EdgeHub](https://kubeedge.io/en/docs/architecture/edge/edgehub): EdgeHub 是一个 Web Socket 客户端，负责与边缘计算的云服务（例如 KubeEdge 架构图中的 Edge Controller）交互，包括同步云端资源更新、报告边缘主机和设备状态变化到云端等功能。
- [Edged](https://kubeedge.io/en/docs/architecture/edge/edged): Edged 是运行在边缘节点的代理，用于管理容器化的应用程序。
- [EventBus](https://kubeedge.io/en/docs/architecture/edge/eventbus): EventBus 是一个与 MQTT 服务器 (mosquitto) 交互的 MQTT 客户端，为其他组件提供订阅和发布功能。
- [ServiceBus](https://kubeedge.io/en/docs/architecture/edge/servicebus): ServiceBus 是一个运行在边缘的 HTTP 客户端，接受来自云上服务的请求，与运行在边缘端的 HTTP 服务器交互，提供了云上服务通过 HTTP 协议访问边缘端 HTTP 服务器的能力。
- [DeviceTwin](https://kubeedge.io/en/docs/architecture/edge/devicetwin): DeviceTwin 负责存储设备状态并将设备状态同步到云，它还为应用程序提供查询接口。
- [MetaManager](https://kubeedge.io/en/docs/architecture/edge/metamanager): MetaManager 是消息处理器，位于 Edged 和 Edgehub 之间，它负责向轻量级数据库 (SQLite) 存储/检索元数据。

## 兼容性

### Kubernetes 版本兼容

|                        | Kubernetes 1.16 | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20 | Kubernetes 1.21 | Kubernetes 1.22 |
|------------------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|
| KubeEdge 1.10          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge 1.11          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge 1.12          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge HEAD (master) | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |

说明：
* `✓` KubeEdge 和 Kubernetes 的版本是完全兼容的
* `+` KubeEdge 中有些特性或 API 对象可能在对应的 Kubernetes 版本中不存在
* `-` Kubernetes 中有些特性或 API 对象可能在对应的 KubeEdge 版本中不可用

## 使用

从此[文档](https://kubeedge.io/en/docs)开始你的 KubeEdge 之旅！

有关更多详细信息，请参阅我们在 [kubeedge.io](https://kubeedge.io) 上的文档。

要深入了解 KubeEdge，请在 [examples](https://github.com/kubeedge/examples) 中尝试一些示例。

## 路线图

* [2021 Roadmap](./docs/roadmap.md#roadmap)

## 社区例会

例会时间：
- 欧洲时间：**北京时间 周三 16:30-17:30**（每双周一次，从 2020 年 2 月 19 日开始）。[『查询本地时间』](https://www.thetimezoneconverter.com/?t=16%3A30&tz=GMT%2B8&)
- 太平洋时间：**北京时间 周三 10:00-11:00**（每双周一次，从 2020 年 2 月 26 日开始）。[『查询本地时间』](https://www.thetimezoneconverter.com/?t=10%3A00&tz=GMT%2B8&)

会议资源：
- [会议纪要和议程](https://docs.google.com/document/d/1Sr5QS_Z04uPfRbA7PrXr3aPwCRpx7EtsyHq7mp6CnHs/edit)
- [会议视频记录](https://www.youtube.com/playlist?list=PLQtlO1kVWGXkRGkjSrLGEPJODoPb8s5FM)
- [会议链接](https://zoom.us/j/4167237304)
- [会议日历](https://calendar.google.com/calendar/embed?src=8rjk8o516vfte21qibvlae3lj4%40group.calendar.google.com) | [订阅日历](https://calendar.google.com/calendar?cid=OHJqazhvNTE2dmZ0ZTIxcWlidmxhZTNsajRAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ)

## 支持

如果您需要支持，请从[故障排除指南](https://kubeedge.io/en/docs/developer/troubleshooting)开始，然后按照我们概述的流程进行操作。

如果您有任何疑问，请以下方式与我们联系：

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)
- [slack](https://join.slack.com/t/kubeedge/shared_invite/enQtNjc0MTg2NTg2MTk0LWJmOTBmOGRkZWNhMTVkNGU1ZjkwNDY4MTY4YTAwNDAyMjRkMjdlMjIzYmMxODY1NGZjYzc4MWM5YmIxZjU1ZDI)
- [twitter](https://twitter.com/kubeedge)

## 贡献

如果您有兴趣成为一个贡献者，也想参与到 KubeEdge 的代码开发中，请查看 [CONTRIBUTING](./CONTRIBUTING.md) 获取更多关于如何提交 Patch 和贡献的流程。

## 安全

### 安全审计报告

KubeEdge的第三方安全审计报告已于2022年7月完成。此外，KubeEdge社区对KubeEdge进行了系统的安全分析和威胁建模。详细报告如下。

- [安全审计报告](https://github.com/kubeedge/community/blob/master/sig-security/sig-security-audit/KubeEdge-security-audit-2022.pdf)

- [威胁建模及安全防护分析白皮书](https://github.com/kubeedge/community/blob/master/sig-security/sig-security-audit/KubeEdge-threat-model-and-security-protection-analysis.md)

### 报告安全漏洞

我们鼓励漏洞研究人员和行业组织主动将KubeEdge社区的疑似安全漏洞报告给KubeEdge社区安全团队(`cncf-kubeedge-security@lists.cncf.io`)。我们会快速的响应、分析和解决上报的安全问题或安全漏洞。
详细漏洞处理流程及如何上报漏洞请查看 [Security Policy](https://github.com/kubeedge/community/blob/master/team-security/SECURITY.md)。

## 许可证

KubeEdge 基于 Apache 2.0 许可证，查看 [LICENSE](./LICENSE) 获取更多信息。
