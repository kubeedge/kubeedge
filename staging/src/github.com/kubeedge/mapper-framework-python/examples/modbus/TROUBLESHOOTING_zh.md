# Modbus RTU Mapper 排错指南

本文记录工业温控器示例从 HTTP `404` 到成功连接 Modbus 模拟器时遇到的完整
故障链。排查时应从云端资源、CloudCore、EdgeCore/DMI、Mapper、串口五层依次
确认，避免一开始就把 DMI 下发问题误判为 Modbus 通信问题。

## 1. 快速判断故障所在层

| 现象或日志 | 含义 | 排查方向 |
| --- | --- | --- |
| `/api/v1/ping` 返回 200 | Mapper HTTP Server 正常 | 不能证明 Device 已注册 |
| API 返回 `404` 和 `'default/industrial-thermostat'` | Mapper 中没有该 Device 的活动会话 | 检查 DMI 下发或 `DEVICE_START_PENDING` |
| `MAPPER_REGISTER_RESPONSE {}` | EdgeCore DMI 返回空 Device/DeviceModel 列表 | 检查 CloudCore 下发和 EdgeCore DMI 缓存 |
| 响应包含 `deviceList`、`modelList` | 云端到 EdgeCore 的资源链路已打通 | 继续检查设备初始化 |
| `DEVICE_START_PENDING` | Device 已下发，但驱动初始化失败 | 直接读取事件中的 `error` |
| `REGISTER_DEVICE` 或 `DEVICE_RECONNECTED` | 驱动初始化成功 | 可以调用 HTTP 读写 API |

HTTP 404 中的错误：

```json
{"statusCode": 404, "error": "'default/industrial-thermostat'"}
```

表示框架在 `devices`/`sessions` 中找不到 `default/industrial-thermostat`，不是
`temperature_pv`、`humidity_rh` 或 `alarm_status` 属性名错误。

## 2. 首先确认云端资源

在控制面执行：

```bash
kubectl get nodes -o wide
kubectl get devices.devices.kubeedge.io -A

kubectl get devices.devices.kubeedge.io industrial-thermostat \
  -n default \
  -o jsonpath='{.spec.nodeName}{"\n"}{.spec.protocol.protocolName}{"\n"}{.spec.deviceModelRef.name}{"\n"}'

kubectl get devicemodels.devices.kubeedge.io industrial-thermostat-model \
  -n default
```

预期输出中的三个关键值是：

```text
<实际边缘节点名>
modbus-rtu
industrial-thermostat-model
```

注意：

- `spec.nodeName` 必须使用 `kubectl get nodes` 第一列的节点名，不能只依据主机
  hostname 猜测；
- `spec.protocol.protocolName` 必须和 `mapper-config.yaml` 中的
  `common.protocol` 完全一致；
- Device 和 DeviceModel 必须处于同一 namespace。

## 3. CloudCore 无权访问 DeviceStatus

### 3.1 典型现象

Device、DeviceModel 和节点配置均正确，但 Mapper 注册响应仍为空。检查：

```bash
kubectl get crd devicestatuses.devices.kubeedge.io
kubectl get devicestatuses.devices.kubeedge.io \
  industrial-thermostat -n default
```

如果 CRD 存在而实例不存在，再查看 CloudCore：

```bash
kubectl -n kubeedge logs deployment/cloudcore --since=30m |
  grep -Ei 'DeviceStatus|forbidden|devicecontroller|failed|error'
```

确定性的错误如下：

```text
User "system:serviceaccount:kubeedge:cloudcore" cannot list resource
"devicestatuses" in API group "devices.kubeedge.io" at the cluster scope
```

`devices/status` 与 `devicestatuses` 是不同的 Kubernetes 资源。ClusterRole 只包含
前者时，DeviceStatus informer 仍会因 `forbidden` 无法启动，CloudCore 也无法为
Device 创建 DeviceStatus。

### 3.2 补充最小 RBAC

以下清单假定 CloudCore 使用 namespace `kubeedge` 中名为 `cloudcore` 的
ServiceAccount；不同安装方式应先核对实际名称。

```bash
kubectl apply -f - <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cloudcore-devicestatus
rules:
  - apiGroups: ["devices.kubeedge.io"]
    resources: ["devicestatuses", "devicestatuses/status"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cloudcore-devicestatus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cloudcore-devicestatus
subjects:
  - kind: ServiceAccount
    name: cloudcore
    namespace: kubeedge
EOF
```

验证权限：

```bash
kubectl auth can-i list devicestatuses.devices.kubeedge.io \
  --as=system:serviceaccount:kubeedge:cloudcore
kubectl auth can-i create devicestatuses.devices.kubeedge.io \
  --as=system:serviceaccount:kubeedge:cloudcore
kubectl auth can-i patch devicestatuses.devices.kubeedge.io \
  --as=system:serviceaccount:kubeedge:cloudcore
```

三条命令都应返回 `yes`。然后重启 CloudCore，让 informer 重新处理已有 Device：

```bash
kubectl -n kubeedge rollout restart deployment/cloudcore
kubectl -n kubeedge rollout status deployment/cloudcore
kubectl get devicestatuses.devices.kubeedge.io \
  industrial-thermostat -n default
```

此时 `DeviceStatus` 中出现 `spec: {}`、`status: {}` 是正常的；Mapper 尚未上报前，
状态本来就是空的。

日志中的 metrics tunnel 错误或缺少 `ImagePrePullJob` CRD 是另外的问题，不是本例
HTTP 404 的直接原因。不过它们通常说明 KubeEdge CRD/RBAC 可能安装不完整，应在
联调结束后用相同版本的完整安装清单统一核对。

## 4. EdgeCore SQLite 有设备，但 DMI 注册响应仍为空

在边缘节点检查 EdgeCore 本地数据库：

```bash
sqlite3 /var/lib/kubeedge/edgecore.db \
  "SELECT key,type FROM meta WHERE value LIKE '%industrial-thermostat%';"
```

预期至少包含：

```text
default/devicemodel/industrial-thermostat-model|devicemodel
default/device/industrial-thermostat|device
```

如果记录存在，而 Mapper 的 `MAPPER_REGISTER_RESPONSE` 仍是 `{}`，说明资源已到
EdgeCore，但 DMI 内存缓存没有相应条目。按以下顺序重建缓存：

1. 停止 Mapper；
2. 重启 EdgeCore；
3. 等待 `/etc/kubeedge/dmi.sock` 恢复；
4. 重新启动 Mapper并完成注册。

```bash
systemctl restart edgecore
systemctl is-active edgecore
ls -l /etc/kubeedge/dmi.sock

journalctl -u edgecore --since "2 minutes ago" --no-pager |
  grep -Ei 'dmi worker|init device|device model|DMI Server|failed'
```

应能看到 DMI worker 从数据库初始化 Device 和 DeviceModel。必须重新启动 Mapper，
因为 EdgeCore 重启会替换 Unix Socket，原 Mapper 不会自动完成一次全新的注册。

## 5. `slave_address must be an integer, got 1.0`

### 5.1 根因

KubeEdge 的 `CustomizedValue` 使用 protobuf `Any`。某些 CRD 转换路径会把 YAML
中的普通数字编码为 `FloatValue`，因此 Mapper 实际收到：

```json
"slave_address": 1.0
```

驱动的整数解析器不会把字符串形式的 `1.0` 当作整数，设备因而进入 pending。
同一问题会影响 `baud_rate`、`data_bits`、`stop_bits` 和 `retries`。

### 5.2 修复

在 Device YAML 的 `configData` 中将整数配置显式写为字符串：

```yaml
configData:
  port: /dev/ttyUSB0
  slave_address: "1"
  baud_rate: "9600"
  data_bits: "8"
  parity: none
  stop_bits: "1"
  timeout_seconds: 1.0
  retries: "2"
  probe_on_connect: true
  verify_writes: true
  handle_local_echo: false
```

`timeout_seconds` 本来就是浮点参数，不需要改成整数字符串。修改后重新
`kubectl apply`。在更新传播期间，日志可能先出现一次旧 pending 配置的
`DEVICE_RECONNECT_FAILED`；只要随后出现以下日志，就说明新配置已经生效：

```text
connected device=default/industrial-thermostat ... slave=1 baud=9600 format=8-N-1
```

## 6. 空 `desired: {}` 被误当成写入指令

### 6.1 典型现象

驱动已经打印 `connected`，随后却为每个属性打印空 desired：

```text
CLOUD_DESIRED_VALUE
{
  "property": "temperature_sv",
  "value": ""
}
```

可写的 FLOAT/INT 属性随后可能报：

```text
could not convert string to float: ''
```

设备会话因此再次被移除，HTTP API 仍返回 404。

### 6.2 根因与框架修复

KubeEdge 的 `DeviceProperty.Desired` 是值结构。即使 Device YAML 没有配置 desired，
CRD 到 DMI 的转换也可能生成一个“字段存在但内容为空”的 protobuf 消息。仅使用
`prop.HasField("desired")` 无法判断它是不是实际 desired。

`mapper_framework/models.py` 应只把含 value 或 metadata 的消息视为 desired：

```python
desired_present = prop.HasField("desired") and bool(
    prop.desired.value or prop.desired.metadata
)
```

这样会忽略空 `{}`，同时保留字符串 `"0"` 等有效 desired。应用修复后重启
Mapper；未配置 desired 的设备不应再打印空值的 `CLOUD_DESIRED_VALUE`。

## 7. 串口和 PTY 排查

只有在 Mapper 已收到 Device，并且 `DEVICE_START_PENDING.error` 指向串口或
Modbus 请求时，才进入本层排查。

模拟器使用 `--create-pty` 时会打印两个 PTY：

```text
created PTY pair; set Device protocol.configData.port to /dev/ttys012
simulator started port=/dev/ttys011 ...
```

Device 必须配置日志明确指出的 Mapper 端，而不是 simulator 自己打开的另一端。
Linux PTY 编号会在 simulator 重启后变化，因此每次重启都要重新核对：

```bash
ls -l /dev/pts/<编号>
```

常见错误含义：

- `cannot open Modbus RTU port`：端口不存在、端口选错或权限不足；
- 读取寄存器无响应：simulator 未运行、PTY 端选反、slave address 或 baud rate
  不一致；
- 端口能连接但 probe 失败：检查模拟器是否仍持有另一端 PTY，并确认从站地址。

## 8. 最终验证顺序

Mapper 正常启动时应依次看到：

```text
MAPPER_REGISTER_RESPONSE     # 包含 deviceList/modelList
REGISTER_DEVICE             # 或 DEVICE_RECONNECTED/UPDATE_DEVICE
```

然后执行：

```bash
curl http://127.0.0.1:7777/api/v1/ping
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/temperature_pv
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/humidity_rh
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/alarm_status
```

只有出现 `REGISTER_DEVICE` 或其他成功会话事件后，后三个属性读取才应返回 200。
