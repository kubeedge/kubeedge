# Roadmap

该文档描述了KubeEdge开发的路线图

[在GitHub中定义的里程碑](https://github.com/kubeedge/kubeedge/milestones)代表了最新的计划。

下面的路线图概述了KubeEdge将要添加的新功能。

## 2023 H1

### SIG Node

- 支持在边缘节点使用WasmEdge
- 支持使用Kubectl attach命令访问边缘容器
- 升级Kubernetes依赖版本到v1.24.14, 将边缘默认容器运行时切换成containerd

### SIG Device-IOT

- 提供基于DMI的modbus mapper
- DMI数据面
- 提供支持DMI的mapper框架

### SIG Security

- 为边缘节点上的应用程序提供支持Kube-API端点的身份验证和授权
- 增强边缘明文存储，确保token不落盘

### SIG Scalability

- 支持集群scope资源可靠地分发到边缘节点
- 通过统一通用informer和减少不必要的缓存，将CloudCore内存占用降低了40%

### SIG Networking

- 为edge-tunnel模块新增可配置的TunnelLimitConfig字段
- EdgeMesh容器网络支持CNI特性

### SIG AI

- Sedna
    - 支持非结构化的终身学习
    - 支持未知任务识别
    - 支持展示知识库

- Ianvs
    - 支持整个生命周期的终身学习
    - 提供经典的终身学习测试指标和支持可视化测试结果
    - 提供真实世界的数据集和丰富的终身学习测试示例

### SIG Testing

- 提供硬件兼容性测试套件Provide node conformance test suite
- 提高单元测试覆盖率Improve unit test coverage

### SIG Cluster-Lifecycle

- 提供工具keink，用于使用Docker容器运行本地KubeEdge集群

### UI

- 提供KubeEdge Dashboard的Alpha版本
- 重构KubeEdge官网

## 2023 H2

### SIG Node

- 支持Windows边缘节点
- 对边缘节点进行功能增强，如支持静态Pod、事件报告和可配置的应用迁移策略
- 支持在RTOS系统上运行边缘节点
- 对节点上的设备插件进行改进，如支持多个虚拟GPU
- 支持边缘Serverless
- 远程升级边缘节点特性GA
- 优化节点组功能，如支持更多差异化配置参数

### SIG Device-IOT

- 支持DMI数据平面（H1已完成设计）
- 基于DMI的边缘设备在多节点间的迁移方案
- 更新基于最新DMI框架的内置Mapper
- 多语言Mapper支持调研
- 时序数据库等数据库的集成
- 重构DeviceInstance和Device Model API
- 提高自定义消息传输的可靠性

### SIG Security

- SLSA / CodeQL（要达到SLSA L4仍有一些工作要做）
- Spiffe调研
- 提供通用接口，支持多种加密算法证书

### SIG Scalability

- 集成EdgeMesh规模和性能测试
- 针对IoT设备场景的规模和性能测试

### Stability

- CloudCore的稳定性维护，包括稳定性测试和问题修复
- EdgeMesh稳定性
- 提高云边协同可靠性，如改进边缘 Kube-API接口和logs/exec等稳定性

### SIG Testing

- 单元测试覆盖率提升
- 基于场景的e2e测试用例覆盖率提升
- 集成测试
- 运行时和K8s版本兼容性测试
- Keadm跨版本兼容性测试
- 云-边缘跨版本兼容性测试

### SIG Networking

- 节点离线优化
    - 某节点离线后，其他节点收到ep里对应节点后端被摘除
- 大规模优化
    - 在大规模部署场景中，边缘kube apiserver的负载很高，考虑使用IPVS（IP虚拟服务器）技术来有效处理请求
    - 具有大量服务的情况下，会大大增加了节点上的iptables规则数
    - 容器网络支持CNI特性
- 性能优化：基于eBPF（扩展伯克利数据包过滤器）的内核流量转发
- 分布式消息系统

### SIG Cluster-Lifecycle

- 支持Windows安装和部署
- 支持边缘应用镜像预下载
- 消息路由高可用性（HA）支持

### Docs

- 官网文档优化，包括目录重构以及完备性提升
- 支持更新文档版本
- 完善官网文档，包括DMI开发者指南、对接监控操作指南等
- 在官网案例中心发布案例

### UI

- Dashboard版本迭代
- 在官网新增案例中心
- 在官网新增招聘中心
- 支持官网的版本控制

### SIG AI

- 支持边云协同生命周期学习中的半自动注释
- 支持边云协同生命周期学习中的运行时未知任务处理
- 支持边云协同生命周期学习中的高级离线未知任务处理

### SIG Robotics

- 新增RoboDev仓库：为开发人员更轻松地构建机器人应用程序
- 新增RTF（开箱即用）机器人端到端解决方案：Teleoperation（遥操作）和RoboPilot（具身智能框架）

### Experience

- Example库增强
- 上线到Killer-Coda