# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)


![logo](./docs/images/KubeEdge_logo.png)

KubeEdge 是一个开源的系统，可将本机容器化应用编排和管理扩展到边缘端设备。 它基于Kubernetes构建，为网络和应用程序提供核心基础架构支持，并在云端和边缘端部署应用，同步元数据。KubeEdge 还支持 **MQTT** 协议，允许开发人员编写客户逻辑，并在边缘端启用设备通信的资源约束。KubeEdge 包含云端和边缘端两部分。

注意：
1.3以前的版本不再支持，请尝试升级到支持版本。

## 优势

### 边缘计算

通过在边缘端运行业务逻辑，可以在本地保护和处理大量数据。KubeEdge 减少了边和云之间的带宽请求，加快响应速度，并保护客户数据隐私。

### 简化开发

开发人员可以编写常规的基于 http 或 mqtt 的应用程序，容器化并在边缘或云端任何地方运行。

### Kubernetes 原生支持

使用 KubeEdge 用户可以在边缘节点上编排应用、管理设备并监控应用程序/设备状态，就如同在云端操作 Kubernetes 集群一样。

### 丰富的应用程序

用户可以轻松地将复杂的机器学习、图像识别、事件处理等高层应用程序部署到边缘端。

## 介绍

KubeEdge 由以下组件构成:

### 云上部分
- [CloudHub](https://kubeedge.io/en/docs/architecture/cloud/cloudhub): CloudHub 是一个 Web Socket 服务端，负责监听云端的变化, 缓存并发送消息到 EdgeHub。
- [EdgeController](https://kubeedge.io/en/docs/architecture/cloud/edge_controller): EdgeController 是一个扩展的 Kubernetes 控制器，管理边缘节点和 Pods 的元数据确保数据能够传递到指定的边缘节点。
- [DeviceController](https://kubeedge.io/en/docs/architecture/cloud/device_controller): DeviceController 是一个扩展的 Kubernetes 控制器，管理边缘设备，确保设备信息、设备状态的云边同步。


### 边缘部分
- [EdgeHub](https://kubeedge.io/en/docs/architecture/edge/edgehub): EdgeHub 是一个 Web Socket 客户端，负责与边缘计算的云服务（例如 KubeEdge 架构图中的 Edge Controller）交互，包括同步云端资源更新、报告边缘主机和设备状态变化到云端等功能。
- [Edged](https://kubeedge.io/en/docs/architecture/edge/edged): Edged 是运行在边缘节点的代理，用于管理容器化的应用程序。
- [EventBus](https://kubeedge.io/en/docs/architecture/edge/eventbus): EventBus 是一个与 MQTT 服务器（mosquitto）交互的 MQTT 客户端，为其他组件提供订阅和发布功能。
- [ServiceBus](https://kubeedge.io/en/docs/architecture/edge/servicebus): ServiceBus是一个运行在边缘的HTTP客户端，接受来自云上服务的请求，与运行在边缘端的HTTP服务器交互，提供了云上服务通过HTTP协议访问边缘端HTTP服务器的能力。
- [DeviceTwin](https://kubeedge.io/en/docs/architecture/edge/devicetwin): DeviceTwin 负责存储设备状态并将设备状态同步到云，它还为应用程序提供查询接口。
- [MetaManager](https://kubeedge.io/en/docs/architecture/edge/metamanager): MetaManager 是消息处理器，位于 Edged 和 Edgehub 之间，它负责向轻量级数据库（SQLite）存储/检索元数据。


### 架构

![架构图](docs/images/kubeedge_arch.png)

## 兼容性

### Kubernetes 版本兼容

|                     | Kubernetes 1.13 | Kubernetes 1.14 | Kubernetes 1.15 | Kubernetes 1.16 | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 |
|---------------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|
| KubeEdge 1.3        | ✓               | ✓              | ✓               | ✓               | ✓               | ✓               | ✓              |
| KubeEdge 1.4        | ✓               | ✓              | ✓               | ✓               | ✓              | ✓               | ✓               |
| KubeEdge 1.5        | ✓               | ✓              | ✓               | ✓               | ✓              | ✓               | ✓               |
| KubeEdge HEAD       | ✓               | ✓              | ✓               | ✓               | ✓              | ✓               | ✓               |

说明:
* `✓` KubeEdge和Kubernetes的版本是完全兼容的
* `+` KubeEdge中有些特性或API对象可能在对应的Kubernetes版本中不存在
* `-` Kubernetes中有些特性或API对象可能在对应的KubeEdge版本中不可用

### Golang 版本依赖

|                         | Golang 1.11     | Golang 1.12     | Golang 1.13     | Golang 1.14     |
|-------------------------|-----------------|-----------------|-----------------|-----------------|
| KubeEdge 1.2            | ✗               | ✓               | ✓               | ✓               |
| KubeEdge 1.3            | ✗               | ✓               | ✓               | ✓               |
| KubeEdge 1.4            | ✗               | ✗               | ✗               | ✓               |
| KubeEdge HEAD (master)  | ✗               | ✗               | ✗               | ✓               |

## 使用

* [快速部署](https://kubeedge.io/en/docs/setup/keadm)

## 路线图

* [2020 Q2 Roadmap](./docs/roadmap_zh.md#2020-q2-roadmap)

## 社区例会

例会时间：
- 欧洲时间：**北京时间 周三 16:30-17:30** (每双周一次，从2020年2月19日开始)。
([查询本地时间](https://www.thetimezoneconverter.com/?t=16%3A30&tz=GMT%2B8&))
- 太平洋时间：**北京时间 周三 10:00-11:00** (每双周一次，从2020年2月26日开始)。
([查询本地时间](https://www.thetimezoneconverter.com/?t=10%3A00&tz=GMT%2B8&))

会议资源：
- [会议纪要和议程](https://docs.google.com/document/d/1Sr5QS_Z04uPfRbA7PrXr3aPwCRpx7EtsyHq7mp6CnHs/edit)
- [会议视频记录](https://www.youtube.com/playlist?list=PLQtlO1kVWGXkRGkjSrLGEPJODoPb8s5FM)
- [会议链接](https://zoom.us/j/4167237304)
- [会议日历](https://calendar.google.com/calendar/embed?src=8rjk8o516vfte21qibvlae3lj4%40group.calendar.google.com) | [订阅日历](https://calendar.google.com/calendar?cid=OHJqazhvNTE2dmZ0ZTIxcWlidmxhZTNsajRAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ)

## 文档

从此[文档](https://kubeedge.io/en/docs)开始你的KubeEdge之旅！
访问[https://docs.kubeedge.io](https://docs.kubeedge.io) 获得更多详细信息。
一些说明 KubeEdge 平台的使用案例的示例应用程序和演示可以在[这个仓库](https://github.com/kubeedge/examples) 中找到。

## 支持

如果您需要支持，请从 [故障排除指南](https://kubeedge.io/en/docs/developer/troubleshooting) 开始，然后按照我们概述的流程进行操作。

如果您有任何疑问，请以下方式与我们联系：

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)
- [slack](https://join.slack.com/t/kubeedge/shared_invite/enQtNjc0MTg2NTg2MTk0LWJmOTBmOGRkZWNhMTVkNGU1ZjkwNDY4MTY4YTAwNDAyMjRkMjdlMjIzYmMxODY1NGZjYzc4MWM5YmIxZjU1ZDI)
- [twitter](https://twitter.com/kubeedge)

## 贡献

如果您有兴趣成为一个贡献者，也想参与到KubeEdge的代码开发中，
请查看[CONTRIBUTING](./CONTRIBUTING.md)获取更多关于如何提交Patch和贡献的流程。

## 许可证

KubeEdge基于Apache 2.0许可证，查看[LICENSE](./LICENSE)获取更多信息。
