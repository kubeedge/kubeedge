# Roadmap

该文档描述了KubeEdge的开发计划

[milestones defined in GitHub](https://github.com/kubeedge/kubeedge/milestones) 包含了最新的开发进度

KubeEdge 1.1 版本是我们目前的稳定版本， 在2019年Q4将添加如下的新特性。

## 2019 Q4 Roadmap

- 云上组件Cloudcore支持多实例部署（HA）
- 支持针对边缘应用的exec、logs API
- 支持云边消息的可靠传输
- 云边消息的传输支持使用protobuf格式
- 完成性能、规模测试，并输出测试报告
- 支持在云端管理边缘集群（EdgeSite等）
- 持续提升KubeEdge的性能与可靠性
- 支持边缘应用的Ingress访问
- 升级Kubernetes依赖到1.16版本
- 提升贡献者体验，提供治理策略、Release计划、社区成员角色管理等
- 提升KubeEdge的安装、使用体验
- 添加更多的文档，将文档从主库移到Website库

## Future

- 支持云边应用通过edgemesh通信
- 在边缘端提供函数即服务（Function as a Service，FaaS）
- 在边缘端节点支持更多类型的设备协议，如 OPC-UA, Zigbee等
- 评估并启用具有数千个边缘节点和数百万设备的超大规模边缘集群
- 启用应用的智能调度，扩大边缘节点的规模
- 提供边缘端的应用、节点监控能力
- 提升边缘端的安全能力
- 评估使用gRPC协议进行云边通信