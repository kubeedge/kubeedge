# CloudHub

## CloudHub 概述

CloudHub是cloudcore的一个模块，介于Controller和Edge端之间。
它同时支持基于websocket的连接以及[QUIC](https://quicwg.org/ops-drafts/draft-ietf-quic-applicability.html)协议访问。Edgehub可以选择任一协议来访问Cloudhub。
CloudHub的功能是保障边缘Edge端与控制器Controller之间的通信


与边缘端的通信（通过EdgeHub模块）通过websocket连接HTTP完成。而在内部通讯方面，它直接与控制器Controller进行通讯。
发送到CloudHub的所有请求都是上下文对象，它与标记为其nodeID的事件对象的映射通道一起存储在channelQ中。

CloudHub执行的主要功能是 :

- 获取消息上下文并为事件创建ChannelQ
- 通过WebSocket创建HTTP连接
- websocket服务连接
- 从边缘端读取消息
- 发送消息到边缘端
- 发送消息到控制器Controller


### 获取消息上下文并为事件创建ChannelQ:

上下文对象存储在channelQ中。
所有的节点ID通道创建的同时，消息会被转换为事件对象，然后事件对象会通过通道进行传递。

### 通过websocket创建http连接:

- TLS证书通过上下文对象中提供的路径加载
- HTTP服务器以TLS配置启动
- 然后将HTTP连接升级为websocket连接，用来接收传输的对象
- ServeConn函数可服务所有传入连接

### 从边缘端读取消息:

- 首先，设置保持活动间隔的最后期限
- 然后读取来自连接的JSON消息
- 设置完消息路由器详细信息之后
- 然后将消息转换为事件对象来进行云内部通信
- 最后，事件消息会被推送给控制器Controller

### 发送消息到边缘端:

- 首先，所有事件对象被接收后会指定nodeID
- 检查是否有相同请求，确保节点的活动性
- 事件对象会被转换为消息结构
- 写入期限已设定。然后将消息传递到websocket

### 发送消息到控制器Controller:

- 每次向Websocket发出请求时，带有时间戳，clientID和事件类型的消息都会默认发送到控制器Controller
- 当节点断开连接后，错误会被抛出，描述节点故障的事件消息也会被发送到控制器Controller

## 用法

可以通过以下三种方式配置 CloudHub :

- **仅启动websocket服务器** ：单击[此处](/docs/proposals/quic-design.md#start-the-websocket-server-only)以查看详细信息。
- **仅启动quic服务器**：单击[此处](/docs/proposals/quic-design.md#start-the-quic-server-only)以查看详细信息。
- **同时启动websocket和quic服务器**：单击[此处](/docs/proposals/quic-design.md#start-the-websocket-and-quic-server-at-the-same-time)以查看详细信息




