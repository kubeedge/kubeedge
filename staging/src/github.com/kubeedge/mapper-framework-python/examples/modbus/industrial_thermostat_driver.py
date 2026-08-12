"""Real Modbus RTU driver for the industrial thermostat in this example.

The mapper framework owns device lifecycle, polling, desired-value delivery and
DMI reporting.  This module only adapts BaseDeviceDriver to a PyModbus serial
client and converts the thermostat's 16-bit holding registers.
"""

from __future__ import annotations

import inspect
import json
import logging
import threading
from dataclasses import dataclass
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation, ROUND_HALF_UP
from typing import Any, Callable

from mapper_framework import BaseDeviceDriver, DeviceContext, PropertyContext


LOGGER = logging.getLogger(__name__)


@dataclass(frozen=True)
class RegisterSpec:
    address: int
    data_type: str
    access: str
    scale: Decimal = Decimal("1")
    minimum: Decimal | None = None
    maximum: Decimal | None = None
    values: dict[int, Any] | None = None


REGISTER_SPECS: dict[str, RegisterSpec] = {
    "temperature_pv": RegisterSpec(0x0000, "int16", "read", Decimal("0.1"), Decimal("-20"), Decimal("100")),
    "humidity_rh": RegisterSpec(0x0001, "uint16", "read", Decimal("0.1"), Decimal("0"), Decimal("100")),
    "temperature_sv": RegisterSpec(0x0002, "int16", "read_write", Decimal("0.1"), Decimal("5"), Decimal("35")),
    "power": RegisterSpec(0x0003, "uint16", "read_write", values={0: "OFF", 1: "ON"}),
    "mode": RegisterSpec(
        0x0004,
        "uint16",
        "read_write",
        values={0: "cooling", 1: "heating", 2: "ventilation"},
    ),
    "fan_speed": RegisterSpec(
        0x0006,
        "uint16",
        "read_write",
        values={0: "low", 1: "medium", 2: "high", 3: "auto"},
    ),
    "alarm_status": RegisterSpec(0x000A, "uint16", "read"),
    "modbus_address": RegisterSpec(0x0100, "uint16", "read_write", minimum=Decimal("1"), maximum=Decimal("247")),
    "baud_rate": RegisterSpec(
        0x0101,
        "uint16",
        "read_write",
        values={0: 2400, 1: 4800, 2: 9600, 3: 19200},
    ),
}

ALARM_BITS = {
    0: "over_temperature",
    1: "temperature_sensor_fault",
    2: "humidity_sensor_fault",
    3: "fan_fault",
}


def _as_int(value: Any, field: str) -> int:
    try:
        return int(str(value), 0)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field} must be an integer, got {value!r}") from exc


def _as_bool(value: Any, default: bool = False) -> bool:
    if value is None:
        return default
    if isinstance(value, bool):
        return value
    return str(value).strip().lower() in {"1", "true", "yes", "on"}


class IndustrialThermostatModbusDriver(BaseDeviceDriver):
    """BaseDeviceDriver implementation backed by a real Modbus RTU client."""

    protocol_name = "modbus-rtu"
    client_factory: Callable[..., Any] | None = None

    def __init__(self, device: DeviceContext) -> None:
        super().__init__(device)
        self.client: Any | None = None
        self._lock = threading.RLock()
        self._slave_address = 1
        self._baud_rate = 9600
        self._client_options: dict[str, Any] = {}
        self._last_error = ""
        self._last_success: str | None = None

    def init_device(self) -> None:
        config = self.device.protocol_config
        port = str(config.get("port", "")).strip()
        if not port:
            raise ValueError("protocol.configData.port is required, for example /dev/ttyUSB0")

        self._slave_address = _as_int(config.get("slave_address", 1), "slave_address")
        if not 1 <= self._slave_address <= 247:
            raise ValueError("slave_address must be in range 1..247")

        self._baud_rate = _as_int(config.get("baud_rate", 9600), "baud_rate")
        if self._baud_rate not in {2400, 4800, 9600, 19200}:
            raise ValueError("baud_rate must be one of 2400, 4800, 9600, 19200")

        parity = str(config.get("parity", "N")).strip().upper()
        parity = {"NONE": "N", "EVEN": "E", "ODD": "O"}.get(parity, parity)
        if parity != "N":
            raise ValueError("this thermostat requires parity N/none")

        data_bits = _as_int(config.get("data_bits", 8), "data_bits")
        stop_bits = _as_int(config.get("stop_bits", 1), "stop_bits")
        if data_bits != 8 or stop_bits != 1:
            raise ValueError("this thermostat requires the serial format 8-N-1")

        self._client_options = {
            "port": port,
            "baudrate": self._baud_rate,
            "bytesize": data_bits,
            "parity": parity,
            "stopbits": stop_bits,
            "timeout": float(config.get("timeout_seconds", 1.0)),
            "retries": _as_int(config.get("retries", 2), "retries"),
            "handle_local_echo": _as_bool(config.get("handle_local_echo")),
        }
        self._open_client()

        if _as_bool(config.get("probe_on_connect"), default=True):
            self._read_raw(REGISTER_SPECS["temperature_pv"])
        LOGGER.info(
            "connected device=%s port=%s slave=%d baud=%d format=8-%s-1",
            self.device.id,
            port,
            self._slave_address,
            self._baud_rate,
            parity,
        )

    def get_device_data(self, property_context: PropertyContext) -> Any:
        spec = self._property_spec(property_context)
        raw = self._read_raw(spec)
        value = self._decode_register(raw, spec)
        if property_context.name == "alarm_status" and _as_bool(
            property_context.visitor_config.get("return_alarm_names")
        ):
            return json.dumps(
                [name for bit, name in ALARM_BITS.items() if raw & (1 << bit)],
                ensure_ascii=False,
            )
        return value

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        spec = self._property_spec(property_context)
        if spec.access != "read_write" or property_context.access_mode.lower() == "readonly":
            raise PermissionError(f"property {property_context.name!r} is read-only")

        raw = self._encode_register(value, spec, property_context.name)
        function_code = _as_int(
            property_context.visitor_config.get("write_function_code", "0x06"),
            "write_function_code",
        )
        if function_code not in {0x06, 0x10}:
            raise ValueError("write_function_code must be 0x06 or 0x10")

        with self._lock:
            try:
                self._write_raw(spec.address, raw, function_code)

                # These two device parameters take effect only after the write
                # response. Update the request target before any verification read.
                if property_context.name == "modbus_address":
                    self._slave_address = raw
                elif property_context.name == "baud_rate":
                    self._baud_rate = int(REGISTER_SPECS["baud_rate"].values[raw])
                    self._client_options["baudrate"] = self._baud_rate
                    self._replace_client()

                if _as_bool(self.device.protocol_config.get("verify_writes"), default=True):
                    actual = self._read_raw(spec)
                    if actual != raw:
                        raise IOError(
                            f"write verification failed for {property_context.name}: "
                            f"expected raw {raw}, read raw {actual}"
                        )
            except Exception as exc:
                self._last_error = str(exc)
                raise

    def device_data_write(
        self,
        method_name: str,
        property_context: PropertyContext,
        value: Any,
    ) -> None:
        properties = self.device.methods.get(method_name)
        if properties is None:
            raise ValueError(f"unsupported method: {method_name}")
        if property_context.name not in properties:
            raise ValueError(
                f"method {method_name!r} cannot write property {property_context.name!r}"
            )
        self.set_device_data(value, property_context)

    def get_device_states(self) -> str:
        connected = bool(self.client is not None and getattr(self.client, "connected", False))
        return json.dumps(
            {
                "connected": connected,
                "slaveAddress": self._slave_address,
                "baudRate": self._baud_rate,
                "lastSuccess": self._last_success,
                "lastError": self._last_error or None,
            },
            ensure_ascii=False,
        )

    def stop_device(self) -> None:
        with self._lock:
            if self.client is not None:
                self.client.close()
                self.client = None

    def _open_client(self) -> None:
        factory = self.client_factory
        if factory is None:
            try:
                from pymodbus.client import ModbusSerialClient
            except ImportError as exc:
                raise RuntimeError(
                    "PyModbus is required; install examples/modbus/requirements.txt"
                ) from exc
            factory = ModbusSerialClient
        self.client = factory(**self._client_options)
        if not self.client.connect():
            self.client.close()
            self.client = None
            raise ConnectionError(
                f"cannot open Modbus RTU port {self._client_options['port']!r}"
            )

    def _replace_client(self) -> None:
        if self.client is not None:
            self.client.close()
        self.client = None
        self._open_client()

    def _ensure_connected(self) -> Any:
        if self.client is None:
            self._open_client()
        elif not getattr(self.client, "connected", False) and not self.client.connect():
            raise ConnectionError("Modbus RTU client is disconnected")
        return self.client

    def _read_raw(self, spec: RegisterSpec) -> int:
        with self._lock:
            try:
                client = self._ensure_connected()
                response = client.read_holding_registers(
                    spec.address,
                    count=1,
                    **self._device_id_argument(client.read_holding_registers),
                )
                self._check_response(response, f"read holding register 0x{spec.address:04X}")
                if len(response.registers) != 1:
                    raise IOError(f"expected one register, got {len(response.registers)}")
                self._mark_success()
                return int(response.registers[0]) & 0xFFFF
            except Exception as exc:
                self._last_error = str(exc)
                raise

    def _write_raw(self, address: int, raw: int, function_code: int) -> None:
        try:
            client = self._ensure_connected()
            if function_code == 0x06:
                method = client.write_register
                response = method(
                    address,
                    raw,
                    **self._device_id_argument(method),
                )
            else:
                method = client.write_registers
                response = method(
                    address,
                    [raw],
                    **self._device_id_argument(method),
                )
            self._check_response(response, f"write holding register 0x{address:04X}")
            self._mark_success()
        except Exception as exc:
            self._last_error = str(exc)
            raise

    def _device_id_argument(self, method: Callable[..., Any]) -> dict[str, int]:
        """Support PyModbus 3.x's slave -> device_id API rename."""
        parameters = inspect.signature(method).parameters
        key = "device_id" if "device_id" in parameters else "slave"
        return {key: self._slave_address}

    @staticmethod
    def _check_response(response: Any, operation: str) -> None:
        if response is None:
            raise IOError(f"{operation} returned no response")
        if response.isError():
            exception_code = getattr(response, "exception_code", None)
            suffix = f", exception code={exception_code}" if exception_code is not None else ""
            raise IOError(f"{operation} failed: {response}{suffix}")

    def _property_spec(self, property_context: PropertyContext) -> RegisterSpec:
        try:
            spec = REGISTER_SPECS[property_context.name]
        except KeyError as exc:
            raise KeyError(f"unknown thermostat property {property_context.name!r}") from exc

        visitor = property_context.visitor_config
        register_type = str(visitor.get("register_type", "holding_register")).lower()
        if register_type != "holding_register":
            raise ValueError(f"{property_context.name}: only holding_register is supported")
        if "address" in visitor:
            configured = _as_int(visitor["address"], "address")
            if configured != spec.address:
                raise ValueError(
                    f"{property_context.name}: configured address 0x{configured:04X} "
                    f"does not match device register 0x{spec.address:04X}"
                )
        return spec

    @staticmethod
    def _decode_register(raw: int, spec: RegisterSpec) -> int | float:
        if spec.data_type == "int16" and raw & 0x8000:
            raw -= 0x10000
        scaled = Decimal(raw) * spec.scale
        return int(scaled) if scaled == scaled.to_integral_value() else float(scaled)

    @staticmethod
    def _encode_register(value: Any, spec: RegisterSpec, property_name: str) -> int:
        if spec.values:
            for raw, label in spec.values.items():
                if str(value).strip().lower() == str(label).strip().lower():
                    value = raw
                    break
        try:
            physical = Decimal(str(value))
        except (InvalidOperation, ValueError) as exc:
            raise ValueError(f"{property_name}: invalid value {value!r}") from exc

        if spec.minimum is not None and physical < spec.minimum:
            raise ValueError(f"{property_name}: {physical} is below minimum {spec.minimum}")
        if spec.maximum is not None and physical > spec.maximum:
            raise ValueError(f"{property_name}: {physical} is above maximum {spec.maximum}")

        raw_decimal = physical / spec.scale
        raw = int(raw_decimal.to_integral_value(rounding=ROUND_HALF_UP))
        if raw_decimal != Decimal(raw):
            raise ValueError(f"{property_name}: {value!r} cannot be represented at scale {spec.scale}")
        if spec.values and raw not in spec.values:
            raise ValueError(
                f"{property_name}: unsupported value {value!r}; allowed values are "
                f"{sorted(spec.values)} or {list(spec.values.values())}"
            )
        if spec.data_type == "int16":
            if not -32768 <= raw <= 32767:
                raise ValueError(f"{property_name}: raw int16 value out of range")
            return raw & 0xFFFF
        if not 0 <= raw <= 0xFFFF:
            raise ValueError(f"{property_name}: raw uint16 value out of range")
        return raw

    def _mark_success(self) -> None:
        self._last_error = ""
        self._last_success = datetime.now(timezone.utc).isoformat()
