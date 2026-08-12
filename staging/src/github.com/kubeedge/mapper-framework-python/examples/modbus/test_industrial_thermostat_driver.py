from __future__ import annotations

import unittest
from types import SimpleNamespace

from mapper_framework import PropertyContext

from .industrial_thermostat_driver import IndustrialThermostatModbusDriver


class FakeResponse:
    def __init__(self, registers=None, error: bool = False) -> None:
        self.registers = [] if registers is None else registers
        self._error = error

    def isError(self) -> bool:  # PyModbus API spelling
        return self._error


class FakeModbusSerialClient:
    def __init__(self, factory: "FakeClientFactory", **options) -> None:
        self.factory = factory
        self.options = options
        self.connected = False
        self.calls = []

    def connect(self) -> bool:
        self.connected = True
        return True

    def close(self) -> None:
        self.connected = False

    def read_holding_registers(self, address, *, count=1, device_id=1):
        self.calls.append(("read", address, count, device_id))
        return FakeResponse([self.factory.registers.get(address, 0)])

    def write_register(self, address, value, *, device_id=1):
        self.calls.append(("write06", address, value, device_id))
        self.factory.registers[address] = value
        return FakeResponse()

    def write_registers(self, address, values, *, device_id=1):
        self.calls.append(("write10", address, list(values), device_id))
        self.factory.registers[address] = values[0]
        return FakeResponse()


class FakeClientFactory:
    def __init__(self) -> None:
        self.registers = {}
        self.clients = []

    def __call__(self, **options):
        client = FakeModbusSerialClient(self, **options)
        self.clients.append(client)
        return client


def property_context(name: str, *, access="ReadWrite", function="0x06") -> PropertyContext:
    addresses = {
        "temperature_pv": "0x0000",
        "temperature_sv": "0x0002",
        "power": "0x0003",
        "fan_speed": "0x0006",
        "modbus_address": "0x0100",
        "baud_rate": "0x0101",
    }
    return PropertyContext(
        name=name,
        access_mode=access,
        visitor_config={
            "register_type": "holding_register",
            "address": addresses[name],
            "write_function_code": function,
        },
    )


class IndustrialThermostatDriverTest(unittest.TestCase):
    def setUp(self) -> None:
        self.factory = FakeClientFactory()
        IndustrialThermostatModbusDriver.client_factory = self.factory
        device = SimpleNamespace(
            id="default/industrial-thermostat",
            protocol_config={
                "port": "/dev/fake-rs485",
                "slave_address": 1,
                "baud_rate": 9600,
                "parity": "none",
                "verify_writes": True,
                "probe_on_connect": True,
            },
            methods={
                "setTemperature": ("temperature_sv",),
                "setPower": ("power",),
                "setFanSpeed": ("fan_speed",),
                "setModbusAddress": ("modbus_address",),
                "setBaudRate": ("baud_rate",),
            },
        )
        self.driver = IndustrialThermostatModbusDriver(device)
        self.driver.init_device()

    def tearDown(self) -> None:
        self.driver.stop_device()
        IndustrialThermostatModbusDriver.client_factory = None

    def test_reads_signed_scaled_temperature_with_function_03(self) -> None:
        self.factory.registers[0x0000] = 0xFF9C  # -100 raw = -10.0 °C
        value = self.driver.get_device_data(
            property_context("temperature_pv", access="ReadOnly")
        )
        self.assertEqual(value, -10)
        self.assertEqual(self.factory.clients[-1].calls[-1], ("read", 0x0000, 1, 1))

    def test_writes_scaled_temperature_with_function_06_and_verifies(self) -> None:
        prop = property_context("temperature_sv")
        self.driver.device_data_write("setTemperature", prop, 26.5)
        calls = self.factory.clients[-1].calls
        self.assertIn(("write06", 0x0002, 265, 1), calls)
        self.assertEqual(calls[-1], ("read", 0x0002, 1, 1))

    def test_function_10_and_enum_label(self) -> None:
        prop = property_context("fan_speed", function="0x10")
        self.driver.device_data_write("setFanSpeed", prop, "auto")
        self.assertIn(("write10", 0x0006, [3], 1), self.factory.clients[-1].calls)

    def test_read_only_and_range_are_rejected_before_write(self) -> None:
        with self.assertRaises(PermissionError):
            self.driver.set_device_data(
                20,
                property_context("temperature_pv", access="ReadOnly"),
            )
        with self.assertRaises(ValueError):
            self.driver.set_device_data(36, property_context("temperature_sv"))

    def test_slave_address_changes_only_after_successful_write(self) -> None:
        self.driver.device_data_write(
            "setModbusAddress",
            property_context("modbus_address"),
            2,
        )
        calls = self.factory.clients[-1].calls
        self.assertIn(("write06", 0x0100, 2, 1), calls)
        self.assertEqual(calls[-1], ("read", 0x0100, 1, 2))

    def test_baud_rate_recreates_client_after_successful_write(self) -> None:
        old_client = self.factory.clients[-1]
        self.driver.device_data_write(
            "setBaudRate",
            property_context("baud_rate"),
            3,
        )
        self.assertIn(("write06", 0x0101, 3, 1), old_client.calls)
        self.assertEqual(len(self.factory.clients), 2)
        self.assertEqual(self.factory.clients[-1].options["baudrate"], 19200)
        self.assertEqual(self.factory.clients[-1].calls[-1], ("read", 0x0101, 1, 1))


if __name__ == "__main__":
    unittest.main()

