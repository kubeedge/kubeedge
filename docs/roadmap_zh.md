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