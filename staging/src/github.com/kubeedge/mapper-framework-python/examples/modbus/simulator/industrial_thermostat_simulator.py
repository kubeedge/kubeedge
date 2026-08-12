"""Modbus RTU slave simulator matching the industrial thermostat register map."""

from __future__ import annotations

import argparse
import asyncio
import logging
import math
import os
import signal
import time
import tty
from dataclasses import dataclass, field
from typing import Any

from pymodbus.constants import ExcCodes
from pymodbus.server import ModbusSerialServer
from pymodbus.simulator import DataType, SimData, SimDevice


LOGGER = logging.getLogger("industrial-thermostat-simulator")

READ_FUNCTION = 0x03
WRITE_FUNCTIONS = {0x06, 0x10}
BAUD_VALUES = {0: 2400, 1: 4800, 2: 9600, 3: 19200}
BAUD_REGISTERS = {value: key for key, value in BAUD_VALUES.items()}

TEMPERATURE_PV = 0x0000
HUMIDITY_RH = 0x0001
TEMPERATURE_SV = 0x0002
POWER = 0x0003
MODE = 0x0004
FAN_SPEED = 0x0006
ALARM_STATUS = 0x000A
MODBUS_ADDRESS = 0x0100
BAUD_RATE = 0x0101

READ_ONLY_REGISTERS = {TEMPERATURE_PV, HUMIDITY_RH, ALARM_STATUS}
WRITABLE_LIMITS = {
    TEMPERATURE_SV: (50, 350),
    POWER: (0, 1),
    MODE: (0, 2),
    FAN_SPEED: (0, 3),
    MODBUS_ADDRESS: (1, 247),
    BAUD_RATE: (0, 3),
}


class PTYBridge:
    """Two connected pseudo-serial ports for local RTU integration tests."""

    def __init__(self) -> None:
        self.simulator_master, simulator_slave = os.openpty()
        self.mapper_master, mapper_slave = os.openpty()
        tty.setraw(simulator_slave)
        tty.setraw(mapper_slave)
        self.simulator_port = os.ttyname(simulator_slave)
        self.mapper_port = os.ttyname(mapper_slave)
        os.close(simulator_slave)
        os.close(mapper_slave)
        os.set_blocking(self.simulator_master, False)
        os.set_blocking(self.mapper_master, False)
        self._loop: asyncio.AbstractEventLoop | None = None

    def start(self, loop: asyncio.AbstractEventLoop) -> None:
        self._loop = loop
        loop.add_reader(
            self.simulator_master,
            self._forward,
            self.simulator_master,
            self.mapper_master,
        )
        loop.add_reader(
            self.mapper_master,
            self._forward,
            self.mapper_master,
            self.simulator_master,
        )

    def close(self) -> None:
        if self._loop is not None:
            self._loop.remove_reader(self.simulator_master)
            self._loop.remove_reader(self.mapper_master)
            self._loop = None
        os.close(self.simulator_master)
        os.close(self.mapper_master)

    @staticmethod
    def _forward(source: int, destination: int) -> None:
        try:
            data = os.read(source, 4096)
            if data:
                os.write(destination, data)
        except (BlockingIOError, OSError):
            return


def _uint16(value: int) -> int:
    return value & 0xFFFF


@dataclass
class ThermostatState:
    """Physical state plus hooks used by the PyModbus SimDevice action."""

    slave_address: int = 1
    baud_rate: int = 9600
    temperature_c: float = 25.0
    humidity_rh: float = 55.0
    setpoint_c: float = 26.0
    power: int = 0
    mode: int = 0
    fan_speed: int = 3
    injected_alarm_mask: int = 0
    ambient_temperature_c: float = 24.0
    _last_update: float = field(default_factory=time.monotonic, init=False)
    _values: dict[int, int] = field(default_factory=dict, init=False)
    _server: Any | None = field(default=None, init=False, repr=False)
    _pending_baud_register: int | None = field(default=None, init=False, repr=False)

    def __post_init__(self) -> None:
        if not 1 <= self.slave_address <= 247:
            raise ValueError("slave address must be in range 1..247")
        if self.baud_rate not in BAUD_REGISTERS:
            raise ValueError("baud rate must be 2400, 4800, 9600, or 19200")
        if not 5.0 <= self.setpoint_c <= 35.0:
            raise ValueError("setpoint must be in range 5.0..35.0 °C")
        if not 0.0 <= self.humidity_rh <= 100.0:
            raise ValueError("humidity must be in range 0..100 %RH")
        if self.power not in {0, 1}:
            raise ValueError("power must be 0 or 1")
        if self.mode not in {0, 1, 2}:
            raise ValueError("mode must be 0, 1, or 2")
        if self.fan_speed not in {0, 1, 2, 3}:
            raise ValueError("fan speed must be 0, 1, 2, or 3")
        if not 0 <= self.injected_alarm_mask <= 0x0F:
            raise ValueError("alarm mask must be in range 0x0..0xF")
        self._sync_values()

    def build_device(self) -> SimDevice:
        """Build an RTU device with only the documented holding registers."""
        blocks = [
            SimData(
                TEMPERATURE_PV,
                values=self._values[TEMPERATURE_PV],
                datatype=DataType.REGISTERS,
                readonly=True,
            ),
            SimData(
                HUMIDITY_RH,
                values=self._values[HUMIDITY_RH],
                datatype=DataType.REGISTERS,
                readonly=True,
            ),
            SimData(
                TEMPERATURE_SV,
                values=self._values[TEMPERATURE_SV],
                datatype=DataType.REGISTERS,
            ),
            SimData(POWER, values=self._values[POWER], datatype=DataType.REGISTERS),
            SimData(MODE, values=self._values[MODE], datatype=DataType.REGISTERS),
            SimData(
                FAN_SPEED,
                values=self._values[FAN_SPEED],
                datatype=DataType.REGISTERS,
            ),
            SimData(
                ALARM_STATUS,
                values=self._values[ALARM_STATUS],
                datatype=DataType.REGISTERS,
                readonly=True,
            ),
            SimData(
                MODBUS_ADDRESS,
                values=self._values[MODBUS_ADDRESS],
                datatype=DataType.REGISTERS,
            ),
            SimData(
                BAUD_RATE,
                values=self._values[BAUD_RATE],
                datatype=DataType.REGISTERS,
            ),
        ]
        return SimDevice(id=self.slave_address, simdata=blocks, action=self.register_action)

    def attach_server(self, server: ModbusSerialServer) -> None:
        self._server = server

    async def register_action(
        self,
        function_code: int,
        start_address: int,
        address: int,
        count: int,
        current_registers: list[int],
        set_values: list[int] | list[bool] | None,
    ) -> ExcCodes | None:
        """Update live values and enforce the thermostat's register contract."""
        if function_code not in {READ_FUNCTION, *WRITE_FUNCTIONS}:
            return ExcCodes.ILLEGAL_FUNCTION

        self._advance_physics()
        self._copy_state_to_runtime(start_address, current_registers)

        if set_values is None:
            return None
        if function_code not in WRITE_FUNCTIONS or len(set_values) != count:
            return ExcCodes.ILLEGAL_FUNCTION

        pending: list[tuple[int, int]] = []
        for offset, value in enumerate(set_values):
            register = address + offset
            if register in READ_ONLY_REGISTERS or register not in WRITABLE_LIMITS:
                return ExcCodes.ILLEGAL_ADDRESS
            raw = int(value)
            minimum, maximum = WRITABLE_LIMITS[register]
            if not minimum <= raw <= maximum:
                return ExcCodes.ILLEGAL_VALUE
            pending.append((register, raw))

        for register, raw in pending:
            self._apply_write(register, raw)
            runtime_index = register - start_address
            if 0 <= runtime_index < len(current_registers):
                current_registers[runtime_index] = raw
        return None

    def register_image(self) -> list[int]:
        """Return a 0..0x0101 image for tests and diagnostics."""
        image = [0] * (BAUD_RATE + 1)
        for address, value in self._values.items():
            image[address] = value
        return image

    def snapshot(self) -> dict[str, Any]:
        return {
            "slave_address": self.slave_address,
            "baud_rate": self.baud_rate,
            "temperature_pv": round(self.temperature_c, 1),
            "humidity_rh": round(self.humidity_rh, 1),
            "temperature_sv": round(self.setpoint_c, 1),
            "power": self.power,
            "mode": self.mode,
            "fan_speed": self.fan_speed,
            "alarm_status": self._values[ALARM_STATUS],
        }

    def _advance_physics(self) -> None:
        now = time.monotonic()
        elapsed = min(max(now - self._last_update, 0.0), 30.0)
        self._last_update = now

        if self.power and self.mode == 1:  # heating
            target = max(self.setpoint_c, self.ambient_temperature_c)
        elif self.power and self.mode == 0:  # cooling
            target = min(self.setpoint_c, self.ambient_temperature_c)
        else:  # power off or ventilation
            target = self.ambient_temperature_c

        fan_factor = (0.45, 0.7, 1.0, 0.8)[self.fan_speed]
        rate = 0.18 * fan_factor if self.power else 0.04
        difference = target - self.temperature_c
        maximum_step = rate * elapsed
        self.temperature_c += max(-maximum_step, min(maximum_step, difference))

        humidity_target = 55.0 + 3.0 * math.sin(now / 30.0)
        humidity_step = min(elapsed * 0.15, abs(humidity_target - self.humidity_rh))
        if humidity_target < self.humidity_rh:
            humidity_step = -humidity_step
        self.humidity_rh = min(100.0, max(0.0, self.humidity_rh + humidity_step))
        self._sync_values()

    def _sync_values(self) -> None:
        automatic_alarm = 0x01 if self.temperature_c > 90.0 else 0
        self._values.update(
            {
                TEMPERATURE_PV: _uint16(round(self.temperature_c * 10)),
                HUMIDITY_RH: _uint16(round(self.humidity_rh * 10)),
                TEMPERATURE_SV: _uint16(round(self.setpoint_c * 10)),
                POWER: self.power,
                MODE: self.mode,
                FAN_SPEED: self.fan_speed,
                ALARM_STATUS: automatic_alarm | self.injected_alarm_mask,
                MODBUS_ADDRESS: self.slave_address,
                BAUD_RATE: (
                    self._pending_baud_register
                    if self._pending_baud_register is not None
                    else BAUD_REGISTERS[self.baud_rate]
                ),
            }
        )

    def _copy_state_to_runtime(self, start_address: int, registers: list[int]) -> None:
        for address, value in self._values.items():
            index = address - start_address
            if 0 <= index < len(registers):
                registers[index] = value

    def _apply_write(self, register: int, raw: int) -> None:
        self._values[register] = raw
        if register == TEMPERATURE_SV:
            self.setpoint_c = raw / 10.0
        elif register == POWER:
            self.power = raw
        elif register == MODE:
            self.mode = raw
        elif register == FAN_SPEED:
            self.fan_speed = raw
        elif register == MODBUS_ADDRESS:
            self._change_slave_address(raw)
        elif register == BAUD_RATE:
            self._schedule_baud_change(raw)
        LOGGER.info("write register=0x%04X raw=%d state=%s", register, raw, self.snapshot())

    def _change_slave_address(self, new_address: int) -> None:
        old_address = self.slave_address
        self.slave_address = new_address
        self._values[MODBUS_ADDRESS] = new_address
        if self._server is None:
            return
        # PyModbus FC06 reads the stored value once more while constructing its
        # response. Defer the routing-table change until that coroutine finishes.
        self._server.loop.call_later(
            0.001,
            self._apply_slave_address,
            old_address,
            new_address,
        )

    def _apply_slave_address(self, old_address: int, new_address: int) -> None:
        context = getattr(self._server, "context", None)
        devices = getattr(context, "devices", None)
        if isinstance(devices, dict) and old_address in devices:
            devices[new_address] = devices.pop(old_address)
        LOGGER.warning(
            "slave address changed after response: %d -> %d",
            old_address,
            new_address,
        )

    def _schedule_baud_change(self, baud_register: int) -> None:
        new_baud = BAUD_VALUES[baud_register]
        self._pending_baud_register = baud_register
        self._values[BAUD_RATE] = baud_register
        if self._server is None:
            self.baud_rate = new_baud
            self._pending_baud_register = None
            return
        # At 2400 bit/s an 8-byte response takes about 34 ms. Waiting 50 ms
        # preserves the device contract: the old speed is used for the response.
        self._server.loop.call_later(0.05, self._apply_baud_change, new_baud)

    def _apply_baud_change(self, new_baud: int) -> None:
        old_baud = self.baud_rate
        self.baud_rate = new_baud
        self._pending_baud_register = None
        if self._server is not None:
            self._server.comm_params.baudrate = new_baud
            transport = getattr(self._server, "transport", None)
            serial_port = getattr(transport, "sync_serial", None)
            if serial_port is not None:
                serial_port.baudrate = new_baud
        LOGGER.warning("baud rate changed after response: %d -> %d", old_baud, new_baud)


def build_argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Industrial thermostat Modbus RTU simulator"
    )
    port_group = parser.add_mutually_exclusive_group(required=True)
    port_group.add_argument("--port", help="simulator side of an existing PTY/serial pair")
    port_group.add_argument(
        "--create-pty",
        action="store_true",
        help="create and bridge two local pseudo-serial ports automatically",
    )
    parser.add_argument("--slave-address", type=int, default=1)
    parser.add_argument(
        "--baud-rate",
        type=int,
        default=9600,
        choices=sorted(BAUD_REGISTERS),
    )
    parser.add_argument("--temperature", type=float, default=25.0, help="initial measured °C")
    parser.add_argument("--humidity", type=float, default=55.0, help="initial %%RH")
    parser.add_argument("--setpoint", type=float, default=26.0, help="initial target °C")
    parser.add_argument("--power", type=int, default=0, choices=(0, 1))
    parser.add_argument("--mode", type=int, default=0, choices=(0, 1, 2))
    parser.add_argument("--fan-speed", type=int, default=3, choices=(0, 1, 2, 3))
    parser.add_argument(
        "--alarm-mask",
        type=lambda value: int(value, 0),
        default=0,
        help="inject alarm bits, for example 0x06",
    )
    parser.add_argument("--debug", action="store_true")
    return parser


async def run_server(args: argparse.Namespace) -> None:
    bridge: PTYBridge | None = None
    if args.create_pty:
        bridge = PTYBridge()
        bridge.start(asyncio.get_running_loop())
        args.port = bridge.simulator_port
        LOGGER.info(
            "created PTY pair; set Device protocol.configData.port to %s",
            bridge.mapper_port,
        )

    state = ThermostatState(
        slave_address=args.slave_address,
        baud_rate=args.baud_rate,
        temperature_c=args.temperature,
        humidity_rh=args.humidity,
        setpoint_c=args.setpoint,
        power=args.power,
        mode=args.mode,
        fan_speed=args.fan_speed,
        injected_alarm_mask=args.alarm_mask,
    )
    server = ModbusSerialServer(
        context=state.build_device(),
        port=args.port,
        baudrate=state.baud_rate,
        bytesize=8,
        parity="N",
        stopbits=1,
        timeout=1,
        ignore_missing_devices=True,
    )
    state.attach_server(server)

    loop = asyncio.get_running_loop()
    for signal_number in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(
                signal_number,
                lambda: asyncio.create_task(server.shutdown()),
            )
        except NotImplementedError:  # pragma: no cover - Windows event loop
            pass

    LOGGER.info(
        "simulator started port=%s slave=%d baud=%d format=8-N-1 state=%s",
        args.port,
        state.slave_address,
        state.baud_rate,
        state.snapshot(),
    )
    try:
        await server.serve_forever()
    finally:
        if bridge is not None:
            bridge.close()


def main() -> None:
    parser = build_argument_parser()
    args = parser.parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.debug else logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    try:
        asyncio.run(run_server(args))
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
