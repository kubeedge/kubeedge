# Industrial Thermostat Modbus RTU Simulator

This program simulates the physical Modbus RTU slave used by the parent
`industrial_thermostat_driver.py` example. It is not a driver mock: it starts a
PyModbus `ModbusSerialServer`; the Mapper still opens a character device through
PySerial, sends RTU requests, and parses actual RTU responses.

## 1. Simulated behavior

The simulator implements:

- Slave address 1, 9600, 8-N-1, and RS485 half-duplex defaults.
- Function codes `0x03`, `0x06`, and `0x10`.
- The nine holding registers from `0x0000` through `0x0101` documented for the
  thermostat, using zero-based addresses.
- int16/uint16 values, 0.1 temperature/humidity scaling, enumerations, and ranges.
- Dynamic PV and humidity, plus effects of power, mode, fan, and setpoint.
- Modbus `Illegal Data Address` for writes to read-only registers and `Illegal Data
  Value` for out-of-range values.
- Exceptions for undefined registers and unsupported function codes.
- Address and baud-rate changes taking effect after the write response.
- Alarm injection; temperature above 90 °C automatically sets alarm `bit0`.

The dynamic-temperature logic is for integration testing, not thermal modeling:

- With `power=0`, temperature slowly returns to 24 °C ambient.
- With `power=1, mode=0`, cooling approaches the lower of setpoint and ambient.
- With `power=1, mode=1`, heating approaches the higher of setpoint and ambient.
- With `mode=2`, ventilation approaches ambient.
- Fan speed controls the rate and humidity fluctuates slowly around 55 %RH.

## 2. Install

Python 3.10+ is required. From the `mapper-framework-python` root:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --prefer-binary --timeout 60 --retries 2
python -m pip install -r examples/modbus/simulator/requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --prefer-binary --timeout 60 --retries 2
```

`simulator/requirements.txt` references its parent `modbus/requirements.txt`, so
the driver and simulator share the tested `>=3.14,<4` version range and PyModbus
`SimData`/`SimDevice` server datastore.

## 3. Quick start: create virtual serial ports automatically

In terminal one, start the simulator:

```bash
python -m examples.modbus.simulator.industrial_thermostat_simulator --create-pty
```

It prints a pair such as:

```text
created PTY pair; set Device protocol.configData.port to /dev/ttys012
simulator started port=/dev/ttys011 slave=1 baud=9600 format=8-N-1 ...
```

Keep it running. Change `examples/cloud/industrial-thermostat-device.yaml`
`configData.port` from `/dev/ttyUSB0` to the Mapper endpoint shown in the first
line (`/dev/ttys012` above), not the simulator's endpoint. In terminal two:

```bash
python -m mapper_framework \
  --config examples/modbus/mapper-config.yaml \
  --driver examples.modbus.industrial_thermostat_driver:IndustrialThermostatModbusDriver
```

When using EdgeCore, apply the DeviceModel and the updated Device from `examples/cloud`
after
confirming both port and `nodeName`:

```bash
kubectl apply -f examples/cloud/industrial-thermostat-devicemodel.yaml
kubectl apply -f examples/cloud/industrial-thermostat-device.yaml
```

## 4. Read/write integration test

```bash
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/temperature_pv
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/humidity_rh
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/alarm_status
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setTemperature/temperature_sv/30.0
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setPower/power/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setMode/mode/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setFanSpeed/fan_speed/2
```

Repeated `temperature_pv` reads should approach 30 °C. To change address to 2 and
baud-rate enum to 3 (19200):

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setModbusAddress/modbus_address/2
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setBaudRate/baud_rate/3
```

After successful requests, driver and simulator use address 2 and 19200. Before
either process restarts, update the Device YAML or each endpoint will revert to its
own configured start values.

## 5. Command-line options

```text
--create-pty              Create and bridge two PTYs (recommended locally)
--port PATH               Existing serial port or simulator-side PTY
--slave-address 1         Initial slave address, 1–247
--baud-rate 9600          2400/4800/9600/19200
--temperature 25.0        Initial PV in °C
--humidity 55.0           Initial humidity in %RH
--setpoint 26.0           Initial SV, 5–35 °C
--power 0                 0=OFF, 1=ON
--mode 0                  0=cooling, 1=heating, 2=ventilation
--fan-speed 3             0=low, 1=medium, 2=high, 3=auto
--alarm-mask 0x00         Inject alarm bits 0–3
--debug                    Enable PyModbus and simulator debug logs
```

For a 95 °C over-temperature condition plus injected temperature/humidity sensor
faults:

```bash
python -m examples.modbus.simulator.industrial_thermostat_simulator \
  --create-pty --temperature 95 --alarm-mask 0x06
```

`alarm_status` is then `0x07`: automatic over-temperature bit0 plus injected bits
1 and 2.

## 6. Existing PTY or serial pair

With `socat`, create stable links yourself:

```bash
socat -d -d \
  pty,raw,echo=0,link=/tmp/thermostat-simulator \
  pty,raw,echo=0,link=/tmp/thermostat-mapper
python -m examples.modbus.simulator.industrial_thermostat_simulator \
  --port /tmp/thermostat-simulator
```

Set Device `configData.port` to `/tmp/thermostat-mapper`. Two cross-connected
USB-UART/RS485 adapters can also be used; avoid incorrectly paralleling two active
A/B drivers and verify shared ground and direction control.

## 7. PTY limitations

PTY mode carries real Modbus RTU bytes but does not model RS485 voltage, termination,
bias, wiring interference, exact baud-rate character timing, A/B reversal, GPIO
direction control, or USB-adapter failures. It validates Mapper, PyModbus,
register, and business logic; use real RS485 hardware for electrical and timing
validation.

## 8. Automated tests

Tests cover state logic and a complete RTU path: they create a PTY pair, start
`ModbusSerialServer`, and use `ModbusSerialClient` for `0x03`, `0x06`, and `0x10`.

```bash
python -m unittest examples.modbus.simulator.test_simulator -v
python -m unittest examples.modbus.test_industrial_thermostat_driver \
  examples.modbus.simulator.test_simulator -v
```
