# Roadmap

该文档描述了KubeEdge的开发计划

[milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) 包含了最新的开发进度

KubeEdge 1.3 版本是我们目前的稳定版本， 在2020年Q2将添加如下的新特性。

## 2020 Q2 Roadmap

- 云上支持Metrics-Server
- 支持Kubernetes的exec API，支持从云上进入边缘容器
- 升级Kubernetes的依赖版本到1.18
- 支持边缘节点证书轮转
- 升级Go语言依赖到1.14
- 支持边缘应用网关，对外暴露边缘服务
- Device CRD升级改进，支持用户扩展自定义设备协议
- 边缘节点跨Subnet通信
- 支持边缘节点上的应用通过edgecore进行List-Watch操作
- 支持从CloudHub端直接收集来自边缘侧的数据
- 提升KubeEdge的安装、使用体验
- 添加更多的文档，将文档从主库移到Website库

## Future

- 提升贡献者体验，提供治理策略、Release计划、社区成员角色管理等
- 支持在云端管理边缘集群（EdgeSite等）
- 云边通信使用protobuf编码方式
- 完成性能、规模测试，并输出测试报告
- 添加更多的文档，将文档从主库移到Website库
- 支持边缘应用的Ingress访问
- 集成基于Istio的服务网格，进行云边、边边通信
- 支持云边应用通过edgemesh通信
- 在边缘端提供函数即服务（Function as a Service，FaaS）
- 评估并启用具有数千个边缘节点和数百万设备的超大规模边缘集群
- 启用应用的智能调度，扩大边缘节点的规模
- 提升边缘端的安全能力
- 评估使用gRPC协议进行云边通信
