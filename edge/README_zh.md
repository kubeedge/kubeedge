本节包含KubeEdge边组件的源代码

## KubeEdge Edge

边缘有六个主要组件。

- EdgeHub: 一个负责与云服务进行边缘计算交互的web socket客户端(就像KubeEdge架构中的边缘控制器)。这包括同步云端资源更新到边缘，并向云报告边缘主机和设备状态更改。

- Edged: 一个在边缘节点上运行并管理容器化应用程序的代理。

- EventBus: 一个与MQTT服务器交互的MQTT客户机，为其他组件提供发布和订阅功能。

- ServiceBus: 一个与HTTP服务器(REST)交互的HTTP客户端，为云组件提供HTTP客户端功能，以到达运行在边缘的HTTP服务器。

- DeviceTwin: 负责存储设备状态，并将设备状态同步到云。它还为应用程序提供查询接口。

- MetaManager: 位于edge和edgehub之间的消息处理器。它还负责向轻量级数据库(SQLite)存储/检索元数据。