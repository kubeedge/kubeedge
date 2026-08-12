# 工业温控器 Modbus RTU Mapper 驱动开发与使用文档

本目录是 `mapper-framework-python` 的真实设备示例。驱动只实现
`BaseDeviceDriver` 的硬件适配接口；设备生命周期、desired 下发、周期采集、
DMI gRPC 和 Twin 上报仍由原 framework 完成。串口协议由 PyModbus 处理，代码
没有重复实现 Modbus RTU 帧或 CRC。

模拟设备的启动和联调说明请参考
[`simulator/README.md`](simulator/README.md)。

## 1. 文件说明

| 文件                                         | 用途                               |
| ------------------------------------------ | -------------------------------- |
| `industrial_thermostat_driver.py`          | 真实的 `BaseDeviceDriver` 实现        |
| [`../cloud/industrial-thermostat-devicemodel.yaml`](../cloud/industrial-thermostat-devicemodel.yaml) | 设备能力、类型、范围和读写权限                  |
| [`../cloud/industrial-thermostat-device.yaml`](../cloud/industrial-thermostat-device.yaml) | 串口配置、寄存器 visitor、采集周期和方法         |
| `mapper-config.yaml`                       | Mapper DMI、Unix Socket 和 HTTP 配置 |
| `requirements.txt`                         | 本驱动额外需要的 PyModbus 串口依赖           |
| `test_industrial_thermostat_driver.py`     | 不连接硬件的驱动转换与调用测试                  |
| [`TROUBLESHOOTING.md`](TROUBLESHOOTING.md) | 云端、DMI、Mapper 与串口的分层排错指南         |

## 2. 设备和接线

设备使用 RS485 两线制、半双工、Modbus RTU，默认通信参数为：

```text
从站地址 1；9600 bit/s；8 数据位；无校验；1 停止位（8-N-1）
```

推荐使用带自动收发方向控制的隔离型 USB-RS485 转换器：

```text
转换器 A/D+  ───── 设备 A/D+
转换器 B/D-  ───── 设备 B/D-
转换器 GND   ───── 设备信号地（按设备手册决定）
```

- 总线两端按现场条件安装 120 Ω 终端电阻，不要在每个节点都装终端电阻。
- A/B 命名在不同厂商间可能相反；持续超时时，在断电和确认手册后检查极性。
- 本示例假定转换器自动控制 TX/RX 方向。需要 GPIO/RTS 手动方向控制的硬件，
  应先在操作系统/转换器层配置，不应在 `BaseDeviceDriver` 中自己拼 RTU 帧。
- Linux 上用 `ls -l /dev/serial/by-id/` 找稳定设备名，生产环境优先使用
  `/dev/serial/by-id/...`，不要依赖可能变化的 `/dev/ttyUSB0`。

检查串口：

```bash
ls -l /dev/serial/by-id/ /dev/ttyUSB* 2>/dev/null
id
```

运行 Mapper 的用户必须能打开串口。常见 Linux 发行版可将该用户加入
`dialout` 组，然后重新登录：

```bash
sudo usermod -aG dialout "${USER}"
```

## 3. 寄存器映射

驱动使用协议的 0 基地址。PLC 地址仅用于与设备手册对照，不能直接传给
PyModbus。例如 PLC `40001` 对应请求地址 `0x0000`。

| 属性               |     请求地址 | PLC 地址 | 原始类型   | 换算/枚举                        | 权限 |     默认值 |
| ---------------- | -------: | -----: | ------ | ---------------------------- | -- | ------: |
| `temperature_pv` | `0x0000` |  40001 | int16  | raw × 0.1 °C                 | 只读 |       — |
| `humidity_rh`    | `0x0001` |  40002 | uint16 | raw × 0.1 %RH                | 只读 |       — |
| `temperature_sv` | `0x0002` |  40003 | int16  | raw × 0.1 °C，5–35            | 读写 | 26.0 °C |
| `power`          | `0x0003` |  40004 | uint16 | 0=OFF，1=ON                   | 读写 |       0 |
| `mode`           | `0x0004` |  40005 | uint16 | 0=制冷，1=制热，2=通风               | 读写 |       0 |
| `fan_speed`      | `0x0006` |  40007 | uint16 | 0=低，1=中，2=高，3=自动             | 读写 |       3 |
| `alarm_status`   | `0x000A` |  40011 | uint16 | bit0..bit3 报警位               | 只读 |       0 |
| `modbus_address` | `0x0100` |  40257 | uint16 | 1–247，响应后生效                  | 读写 |       1 |
| `baud_rate`      | `0x0101` |  40258 | uint16 | 0/1/2/3→2400/4800/9600/19200 | 读写 |       2 |

`alarm_status` 位定义：

```text
bit0 over_temperature
bit1 temperature_sensor_fault
bit2 humidity_sensor_fault
bit3 fan_fault
```

读取使用功能码 `0x03`。写 visitor 默认使用 `0x06`；若现场设备要求使用
`0x10`，把对应属性的 `write_function_code` 改为 `"0x10"`，驱动会调用
PyModbus 的 `write_registers()`。

## 4. 驱动与 framework 的对应关系

```text
Device.protocol.configData
  → init_device() → ModbusSerialClient(port, baudrate, 8-N-1, timeout)

Device.properties[].visitors.configData
  → get_device_data() → read_holding_registers(address, count=1)
  → set_device_data() → write_register()/write_registers() → 写后回读

Device.spec.methods
  → device_data_write() → 校验方法和属性 → set_device_data()

Device 删除/Mapper 退出
  → stop_device() → client.close()
```

驱动为并行采集线程使用 `RLock`，确保同一 RS485 串口上的请求不会交错。
它兼容 PyModbus 3.x 中从站参数由 `slave` 改名为 `device_id` 的变化；本示例
已在 PyModbus 3.14 上执行测试。

温度写入使用十进制定点换算。例如 `26.5 °C` 写为原始寄存器值 `265`；
超范围或无法用 0.1 精度表示的值会在发帧前被拒绝。每次写入默认都会用
`0x03` 回读确认，可通过 Device 的 `verify_writes: false` 关闭。

## 5. 安装和启动

Python 需要 3.10+。

以下命令在 `staging/src/github.com/kubeedge/mapper-framework-python` 执行：

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt
python -m pip install -r examples/modbus/requirements.txt
```

修改 `examples/cloud/industrial-thermostat-device.yaml`：

1. 把 `spec.nodeName` 改为运行 EdgeCore 的节点名；
2. 把 `configData.port` 改为真实串口，推荐 `/dev/serial/by-id/...`；
3. 若设备已经改过地址或波特率，同步修改 `slave_address` 和 `baud_rate`。

在云端应用资源：

```bash
kubectl apply -f examples/cloud/industrial-thermostat-devicemodel.yaml
kubectl apply -f examples/cloud/industrial-thermostat-device.yaml
```

在边缘节点启动 Mapper：

```bash
python -m mapper_framework \
  --config examples/modbus/mapper-config.yaml \
  --driver examples.modbus.industrial_thermostat_driver:IndustrialThermostatModbusDriver
```

成功启动时，driver 会打开串口并默认读取 `0x0000` 探测设备。端口能打开但
从站无响应时，Device 会保持 pending，framework 按
`reconnect_interval_seconds` 重试初始化。

## 6. 真实读取验证

Mapper HTTP API 的读取最终会调用 driver，而不是读取缓存：

```bash
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/temperature_pv
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/humidity_rh
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/alarm_status
```

例如 `temperature_pv` 的设备原始值为 `0x0109`（265）时，API 和云端 Twin
上报值为 `26.5`。

## 7. 真实写入验证

先查看 Device 声明的方法：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat
```

通过 HTTP 写真实寄存器：

```bash
# 26.5 °C → 寄存器 0x0002 写入 265
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setTemperature/temperature_sv/26.5

# 开机、制热、自动风速
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setPower/power/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setMode/mode/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setFanSpeed/fan_speed/3
```

也可由云端添加或更新 desired。示例 Device 故意不预置 desired，避免 Mapper
第一次连到真实设备时覆盖现场设定：

```bash
kubectl patch device industrial-thermostat -n default --type='json' \
  -p='[{"op":"add","path":"/spec/properties/2/desired","value":{"value":"26.5"}}]'
```

属性数组下标可能因 YAML 调整而变化。生产系统更适合读取当前资源后按属性名
生成 patch，避免硬编码下标。

## 8. 修改通信参数的特别处理

`modbus_address` 和 `baud_rate` 都是高风险配置寄存器：

- 写从站地址时，请求用旧地址发送；收到成功响应后，driver 才切换到新地址并回读。
- 写波特率时，请求用旧波特率发送；收到成功响应后，driver 关闭串口，以新波特率
  创建 PyModbus client，再回读确认。
- 不要同时从另一个主站操作同一条 RS485 总线。
- 修改前记录旧值。如果写响应在总线上丢失，设备可能已经切换，而 Mapper 仍会报告
  失败；此时依次用旧/新参数排查。

示例（地址 1 改为 2、波特率枚举 2 改为 3/19200）：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setModbusAddress/modbus_address/2
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setBaudRate/baud_rate/3
```

修改成功后，也要把 `examples/cloud/industrial-thermostat-device.yaml` 的启动配置更新为新值，
否则 Mapper 重启时会先用旧参数连接。

## 9. 故障定位

| 现象                  | 优先检查                                       |
| ------------------- | ------------------------------------------ |
| `Permission denied` | 串口路径、容器设备映射、`dialout` 权限                   |
| 端口打开但请求超时           | A/B 极性、从站地址、波特率、8-N-1、终端电阻                 |
| `Illegal Address`   | visitor 使用了 PLC 40001 而不是 0 基地址 0，或设备型号不一致 |
| CRC/短帧错误            | 接地、屏蔽、线长、干扰、重复从站地址、两个主站冲突                  |
| 写成功但值恢复             | 设备本地控制逻辑、云端 desired 再次下发、写保护               |
| 改地址/波特率后离线          | 分别用变更前后参数测试，并同步更新 Device 配置                |

可临时打开 PyModbus 日志辅助抓取请求错误；生产环境不要长期输出包含现场数据的
DEBUG 日志：

```python
logging.getLogger("pymodbus").setLevel(logging.DEBUG)
```

## 10. 无硬件测试

测试用假的 PyModbus client 验证 driver 调用了 `0x03`、`0x06`、`0x10`，以及
负温度、缩放、范围、只读保护和通信参数切换；它不会打开本机串口：

```bash
python -m unittest examples.modbus.test_industrial_thermostat_driver -v
```

无硬件测试通过不等于现场链路通过。交付前仍应按第 6、7 节在真实 RS485 总线
上逐项读写，并确认设备面板显示和云端 Twin 一致。
