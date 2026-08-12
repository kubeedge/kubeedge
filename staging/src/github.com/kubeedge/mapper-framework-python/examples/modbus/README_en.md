# Industrial Thermostat Modbus RTU Mapper Driver

This directory is a physical-device example for `mapper-framework-python`. Its
driver implements only the `BaseDeviceDriver` hardware adapter. The framework still
handles lifecycle, desired delivery, periodic collection, DMI gRPC, and Twin
reporting. PyModbus handles the serial protocol, including Modbus RTU frames and
CRC.

For instructions on starting and testing with the simulated device, see
[`simulator/README_en.md`](simulator/README_en.md).

## Files

| File | Purpose |
| --- | --- |
| `industrial_thermostat_driver.py` | Physical `BaseDeviceDriver` implementation |
| [`../cloud/industrial-thermostat-devicemodel.yaml`](../cloud/industrial-thermostat-devicemodel.yaml) | Capability, types, ranges, and access |
| [`../cloud/industrial-thermostat-device.yaml`](../cloud/industrial-thermostat-device.yaml) | Serial settings, visitors, cycles, and methods |
| `mapper-config.yaml` | Mapper DMI, Unix socket, and HTTP configuration |
| `requirements.txt` | Additional PyModbus serial dependency |
| `test_industrial_thermostat_driver.py` | Hardware-free driver tests |

## Hardware and wiring

The device uses two-wire, half-duplex RS485 Modbus RTU: slave address 1, 9600 bit/s,
8 data bits, no parity, one stop bit (8-N-1). Use an isolated USB-RS485 adapter with
automatic transmit/receive direction control.

```text
adapter A/D+ ───── device A/D+
adapter B/D- ───── device B/D-
adapter GND  ───── device signal ground (as required by the manual)
```

Install 120 Ω termination only at both bus ends when required. A/B labels can be
reversed by vendor, so check polarity safely if requests time out. Configure
GPIO/RTS direction-control hardware at the OS/adapter layer, not in the driver.
Prefer a stable `/dev/serial/by-id/...` name over `/dev/ttyUSB0`.

```bash
ls -l /dev/serial/by-id/ /dev/ttyUSB* 2>/dev/null
id
sudo usermod -aG dialout "${USER}"
```

The Mapper user must have serial-port access; log in again after modifying groups.

## Register mapping

The driver uses zero-based protocol addresses. PLC `40001` maps to request address
`0x0000`, not `40001`.

| Property | Request | PLC | Conversion/enum | Access |
| --- | ---: | ---: | --- | --- |
| `temperature_pv` | `0x0000` | 40001 | int16 × 0.1 °C | read only |
| `humidity_rh` | `0x0001` | 40002 | uint16 × 0.1 %RH | read only |
| `temperature_sv` | `0x0002` | 40003 | int16 × 0.1 °C, 5–35 | read/write |
| `power` | `0x0003` | 40004 | 0=OFF, 1=ON | read/write |
| `mode` | `0x0004` | 40005 | 0=cool, 1=heat, 2=ventilate | read/write |
| `fan_speed` | `0x0006` | 40007 | 0=low, 1=medium, 2=high, 3=auto | read/write |
| `alarm_status` | `0x000A` | 40011 | alarm bits 0–3 | read only |
| `modbus_address` | `0x0100` | 40257 | 1–247 | read/write |
| `baud_rate` | `0x0101` | 40258 | 0/1/2/3→2400/4800/9600/19200 | read/write |

Alarm bits 0–3 mean over-temperature, temperature-sensor fault,
humidity-sensor fault, and fan fault. Reads use `0x03`; writes use `0x06` unless a
visitor's `write_function_code` is `"0x10"`, which uses `write_registers()`.

## Driver behavior

```text
protocol.configData → init_device() → ModbusSerialClient(...)
visitor.configData → get_device_data() → read_holding_registers()
                   → set_device_data() → write + read-back
Device.spec.methods → device_data_write() → validate → write
Device removal / Mapper exit → stop_device() → client.close()
```

An `RLock` keeps requests from interleaving on the RS485 port. The driver supports
the PyModbus 3.x `slave`/`device_id` parameter change and is tested with 3.14.
`26.5 °C` is written as register `265`; invalid range or precision is rejected
before sending. Writes are verified with a `0x03` read by default; Device
`verify_writes: false` disables verification.

## Install and start

Python 3.10+ is required. From `mapper-framework-python`:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt
python -m pip install -r examples/modbus/requirements.txt
```

Set the Device `spec.nodeName`, physical `configData.port` (prefer
`/dev/serial/by-id/...`), and current `slave_address`/`baud_rate`, then run:

```bash
kubectl apply -f examples/cloud/industrial-thermostat-devicemodel.yaml
kubectl apply -f examples/cloud/industrial-thermostat-device.yaml
python -m mapper_framework \
  --config examples/modbus/mapper-config.yaml \
  --driver examples.modbus.industrial_thermostat_driver:IndustrialThermostatModbusDriver
```

The driver probes `0x0000` at startup. If the port opens but no slave responds, the
Device stays pending and initialization is retried after `reconnect_interval_seconds`.

## Verify reads and writes

```bash
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/temperature_pv
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/humidity_rh
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/alarm_status
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setTemperature/temperature_sv/26.5
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setPower/power/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setMode/mode/1
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat/setFanSpeed/fan_speed/3
```

HTTP reads call the driver, not a cache. Raw temperature `0x0109` (265) reports
`26.5`. The example has no desired value, preventing an initial connection from
overwriting onsite settings. Production tooling should patch desired values by
property name rather than fixed YAML-array indexes.

Changing `modbus_address` or `baud_rate` is high risk: requests use the old setting;
after a successful reply the driver switches settings and reads back to verify.
Record old values, do not run another master on the bus, and update Device startup
configuration after a change.

## Troubleshooting and tests

| Symptom | Check first |
| --- | --- |
| `Permission denied` | Port path, container device mount, `dialout` permissions |
| Port opens but request times out | A/B polarity, address, baud rate, 8-N-1, termination |
| `Illegal Address` | Use zero-based address; verify device model |
| CRC/short frame | Grounding, shielding, interference, duplicate slave, master conflict |
| Value reverts | Local control, cloud desired redelivery, write protection |

Temporary PyModbus logging can assist diagnostics; do not retain DEBUG site data in
production:

```python
logging.getLogger("pymodbus").setLevel(logging.DEBUG)
```

```bash
python -m unittest examples.modbus.test_industrial_thermostat_driver -v
```

This uses a fake client and does not validate onsite wiring. Before delivery, test
each read and write on the physical RS485 bus and verify both panel and cloud Twin.
