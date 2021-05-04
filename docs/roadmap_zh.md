# Roadmap

该文档描述了KubeEdge开发的路线图

[在GitHub中定义的里程碑](https://github.com/kubeedge/kubeedge/milestones)代表了最新的计划。

下面的路线图概述了KubeEdge将要添加的新功能。

## 2021 H1

### 核心框架

#### 边缘List-Watch

- 边缘端支持List-Watch接口，方便边缘组件接入。

#### 云边自定义消息通道

- 支持云端和边缘端之间的自定义消息传输

#### 稳定支持CloudCore多活

- 支持多个CloudCore实例同时稳定运行

#### 第三方CNI集成支持

- 提供flannel、calico等CNI插件的官方集成支持

#### 第三方CSI集成支持

- 提供Rook、OpenEBS等CSI插件的官方集成支持

#### 支持云端管理边缘群集 (aka. EdgeSite)

#### 在边缘端支持 ingress/网关


### 可维护性

#### 部署优化

- 更加简单、便捷的部署（最好一键部署，支持中国镜像）
- Admission Controller自动部署

#### 边缘应用离线迁移时间自动化配置

- 一键修改Default tolerationSeconds

#### 体验良好的中文文档


### IOT 设备管理

#### 设备Mapper框架标准以及框架生成器

- 制定边缘设备Mapper的实施标准

#### 支持更多协议的mapper

- OPC-UA mapper
- ONVIF mapper


### 安全

#### 完成安全漏洞扫描


### 测试

#### 使用更多的度量和场景改进性能和e2e测试


### 边云协同AI

#### 支持 KubeFlow/ONNX/Pytorch/Mindspore等

#### 边云协同训练与推理


### MEC

#### 跨边云服务发现

#### 5G网络能力开放



## 2021 H2

### 核心框架

#### 云边自定义消息通道

- 云边支持CloudEvent消息协议

#### 数据面跨网络通信

- 边缘-边缘 跨网络通信
- 边缘-中心云 跨网络通信

#### 使用标准的istio进行服务治理控制

#### 云边协同监控

- 支持prometheus push gateway
- 数据管理，支持接收遥测数据和边缘分析。


### IOT 设备管理

#### 设备Mapper框架标准以及框架生成器

- 开发Mapper基本框架生成器

#### 支持更多协议的mapper

- GB/T 28181 mapper


### 边云协同AI

#### 边缘智能benchmark


### MEC

#### 云网融合

#### service catalog

#### 应用漫游