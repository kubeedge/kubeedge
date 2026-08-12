# 工业温控器 Modbus RTU 模拟设备

这个程序模拟父目录 `industrial_thermostat_driver.py` 对应的真实 Modbus RTU
从站。它不是对 driver 的 mock：模拟器启动 PyModbus `ModbusSerialServer`，Mapper
仍通过 PySerial 打开字符设备、发送 RTU 请求并解析实际 RTU 响应。

## 1. 模拟内容

模拟器复现以下行为：

- 从站地址 1、9600、8-N-1、RS485 半双工的默认配置；
- 功能码 `0x03`、`0x06` 和 `0x10`；
- 0 基地址 `0x0000`–`0x0101` 中文档定义的九个 holding register；
- int16/uint16、0.1 温湿度缩放、枚举和值域限制；
- PV、湿度的动态变化，以及 power/mode/fan/setpoint 对温度的影响；
- 只读寄存器写入返回 Modbus `Illegal Data Address`；
- 越界值返回 `Illegal Data Value`；
- 未定义寄存器和不支持的功能码返回 Modbus 异常；
- `modbus_address` 和 `baud_rate` 在写响应后生效；
- 报警位注入，温度超过 90 °C 时自动置 `bit0`。

动态温度规则用于联调而非热力学建模：

- `power=0`：温度缓慢回到环境温度 24 °C；
- `power=1, mode=0`：制冷时向设定温度或环境温度的较低值变化；
- `power=1, mode=1`：制热时向设定温度或环境温度的较高值变化；
- `mode=2`：通风，温度向环境温度变化；
- 风速影响变化速率，湿度在 55 %RH 附近缓慢波动。

## 2. 安装

Python 需要 3.10+。在 `mapper-framework-python` 根目录执行：

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --prefer-binary --timeout 60 --retries 2
python -m pip install -r examples/modbus/simulator/requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --prefer-binary --timeout 60 --retries 2
```

`simulator/requirements.txt` 直接引用父目录的 `modbus/requirements.txt`，因此
driver 和 simulator 使用同一个经过测试的版本范围 `>=3.14,<4`，不会分别解析
出不同版本。模拟器使用该版本提供的 `SimData/SimDevice` server datastore。

## 3. 最快启动：自动创建虚拟串口

终端一，在 `mapper-framework-python` 根目录启动模拟器：

```bash
python -m examples.modbus.simulator.industrial_thermostat_simulator \
  --create-pty
```

启动日志会打印 Mapper 应连接的动态串口，例如：

```text
created PTY pair; set Device protocol.configData.port to /dev/ttys012
simulator started port=/dev/ttys011 slave=1 baud=9600 format=8-N-1 ...
```

保持模拟器运行，把 `examples/cloud/industrial-thermostat-device.yaml` 中的：

```yaml
configData:
  port: /dev/ttyUSB0
```

临时改为日志打印的 Mapper 端路径 `/dev/ttys012`。不要填 simulator 自己使用的
另一端路径。

终端二启动 Mapper：

```bash
python -m mapper_framework \
  --config examples/modbus/mapper-config.yaml \
  --driver examples.modbus.industrial_thermostat_driver:IndustrialThermostatModbusDriver
```

如果正在连接 EdgeCore，先在云端应用 `examples/cloud` 中的 DeviceModel 和修改后的 Device (注意确认已修改 port 和 nodeName)：

```bash
kubectl apply -f examples/cloud/industrial-thermostat-devicemodel.yaml
kubectl apply -f examples/cloud/industrial-thermostat-device.yaml
```

## 4. 读写联调

读取模拟设备：

```bash
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/temperature_pv
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/humidity_rh
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/alarm_status
```

执行真实 RTU 写入并由 driver 回读校验：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setTemperature/temperature_sv/30.0
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setPower/power/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setMode/mode/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setFanSpeed/fan_speed/2
```

随后重复读取 `temperature_pv`，可以看到模拟温度逐渐接近 30 °C。

修改地址和波特率：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setModbusAddress/modbus_address/2
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setBaudRate/baud_rate/3
```

这两个请求成功后，driver 和 simulator 会分别切换到地址 2 和 19200。重启 Mapper
或 simulator 前，应同步更新 Device YAML，否则双方会恢复为各自的命令行/配置值。


也可由云端添加或更新 desired。示例 Device 故意不预置 desired，避免 Mapper
第一次连到真实设备时覆盖现场设定：

```bash
kubectl patch device industrial-thermostat -n default --type='json' \
  -p='[{"op":"add","path":"/spec/properties/2/desired","value":{"value":"26.5"}}]'
```

## 5. 启动参数

```text
--create-pty              自动创建并桥接两个 PTY；本地开发推荐
--port PATH               使用现有串口或 PTY 的 simulator 端
--slave-address 1         初始从站地址，范围 1–247
--baud-rate 9600          2400/4800/9600/19200
--temperature 25.0        初始 PV，单位 °C
--humidity 55.0           初始湿度，单位 %RH
--setpoint 26.0           初始 SV，范围 5–35 °C
--power 0                 0=OFF，1=ON
--mode 0                  0=cooling，1=heating，2=ventilation
--fan-speed 3             0=low，1=medium，2=high，3=auto
--alarm-mask 0x00         注入 bit0..bit3 报警
--debug                    打开 PyModbus 和模拟器调试日志
```

例如模拟 95 °C 超温并同时注入温湿度传感器故障：

```bash
python -m examples.modbus.simulator.industrial_thermostat_simulator \
  --create-pty \
  --temperature 95 \
  --alarm-mask 0x06
```

此时 `alarm_status` 为 `0x07`：自动超温 bit0，加上注入的 bit1、bit2。

## 6. 使用已有 PTY 或串口对

如果系统已经安装 `socat`，也可自己创建固定链接：

```bash
socat -d -d \
  pty,raw,echo=0,link=/tmp/thermostat-simulator \
  pty,raw,echo=0,link=/tmp/thermostat-mapper
```

启动模拟器：

```bash
python -m examples.modbus.simulator.industrial_thermostat_simulator \
  --port /tmp/thermostat-simulator
```

Device 的 `configData.port` 使用 `/tmp/thermostat-mapper`。

也可以用两个 USB-UART/RS485 适配器交叉连接，将 simulator 端的真实串口传给
`--port`。务必避免把两个主动驱动 A/B 的接口错误并联，并确认共地和收发方向。

## 7. PTY 模式的限制

PTY 会传递真实的 Modbus RTU 字节帧，但操作系统的伪终端不会真实模拟电气层：

- 不模拟 RS485 电压、终端电阻、偏置和线路干扰；
- 不严格按配置波特率延迟每个字符；
- 不模拟 A/B 反接、收发方向 GPIO 或 USB 转换器故障；
- 波特率寄存器和两端 PySerial 参数会改变，但 PTY 无法验证示波器层面的时序。

因此 PTY 适合验证 Mapper、PyModbus、寄存器和业务逻辑；电气层和真实时序仍需
用实际 RS485 硬件验证。

## 8. 自动测试

测试同时覆盖状态逻辑和完整 RTU 链路。端到端测试会自动创建 PTY 对，启动
`ModbusSerialServer`，再用 `ModbusSerialClient` 执行 `0x03/0x06/0x10`：

```bash
python -m unittest examples.modbus.simulator.test_simulator -v
```

与真实 driver 一起回归：

```bash
python -m unittest \
  examples.modbus.test_industrial_thermostat_driver \
  examples.modbus.simulator.test_simulator \
  -v
```
