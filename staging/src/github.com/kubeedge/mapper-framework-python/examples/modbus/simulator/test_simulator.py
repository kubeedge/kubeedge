from __future__ import annotations

import asyncio
import unittest

from pymodbus.client import ModbusSerialClient
from pymodbus.constants import ExcCodes
from pymodbus.server import ModbusSerialServer

from .industrial_thermostat_simulator import (
    BAUD_RATE,
    MODBUS_ADDRESS,
    PTYBridge,
    TEMPERATURE_PV,
    TEMPERATURE_SV,
    ThermostatState,
)


class ThermostatStateTest(unittest.TestCase):
    def test_register_validation_and_multiple_write(self) -> None:
        state = ThermostatState()
        registers = state.register_image()

        read_only = asyncio.run(
            state.register_action(0x06, 0, TEMPERATURE_PV, 1, registers, [300])
        )
        self.assertEqual(read_only, ExcCodes.ILLEGAL_ADDRESS)

        invalid_setpoint = asyncio.run(
            state.register_action(0x06, 0, TEMPERATURE_SV, 1, registers, [400])
        )
        self.assertEqual(invalid_setpoint, ExcCodes.ILLEGAL_VALUE)

        result = asyncio.run(
            state.register_action(0x10, 0, TEMPERATURE_SV, 3, registers, [275, 1, 1])
        )
        self.assertIsNone(result)
        self.assertEqual(state.setpoint_c, 27.5)
        self.assertEqual(state.power, 1)
        self.assertEqual(state.mode, 1)

    def test_heating_changes_measured_temperature(self) -> None:
        state = ThermostatState(temperature_c=20.0, setpoint_c=30.0, power=1, mode=1)
        registers = state.register_image()
        before = registers[TEMPERATURE_PV]
        state._last_update -= 10.0
        result = asyncio.run(
            state.register_action(0x03, 0, TEMPERATURE_PV, 1, registers, None)
        )
        self.assertIsNone(result)
        self.assertGreater(registers[TEMPERATURE_PV], before)

    def test_alarm_injection(self) -> None:
        state = ThermostatState(injected_alarm_mask=0x06)
        self.assertEqual(state.snapshot()["alarm_status"], 0x06)


class RTUEndToEndTest(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self) -> None:
        self.bridge = PTYBridge()
        self.bridge.start(asyncio.get_running_loop())
        self.state = ThermostatState()
        self.server = ModbusSerialServer(
            context=self.state.build_device(),
            port=self.bridge.simulator_port,
            baudrate=9600,
            bytesize=8,
            parity="N",
            stopbits=1,
            timeout=1,
            ignore_missing_devices=True,
        )
        self.state.attach_server(self.server)
        await self.server.serve_forever(background=True)

    async def asyncTearDown(self) -> None:
        await self.server.shutdown()
        self.bridge.close()

    def _client(self, baudrate: int = 9600) -> ModbusSerialClient:
        client = ModbusSerialClient(
            port=self.bridge.mapper_port,
            baudrate=baudrate,
            bytesize=8,
            parity="N",
            stopbits=1,
            timeout=1,
            retries=1,
        )
        self.assertTrue(client.connect())
        return client

    async def test_function_codes_and_after_response_settings(self) -> None:
        def initial_operations() -> None:
            client = self._client()
            response = client.read_holding_registers(0, count=5, device_id=1)
            self.assertFalse(response.isError())
            self.assertEqual(response.registers[:5], [250, 550, 260, 0, 0])

            response = client.write_register(TEMPERATURE_SV, 265, device_id=1)
            self.assertFalse(response.isError())
            response = client.write_registers(TEMPERATURE_SV, [270, 1, 1], device_id=1)
            self.assertFalse(response.isError())

            response = client.write_register(TEMPERATURE_PV, 300, device_id=1)
            self.assertTrue(response.isError())
            self.assertEqual(response.exception_code, ExcCodes.ILLEGAL_ADDRESS)

            response = client.write_register(MODBUS_ADDRESS, 2, device_id=1)
            self.assertFalse(response.isError())
            response = client.read_holding_registers(MODBUS_ADDRESS, count=1, device_id=2)
            self.assertEqual(response.registers, [2])

            response = client.write_register(BAUD_RATE, 3, device_id=2)
            self.assertFalse(response.isError())
            response = client.read_holding_registers(BAUD_RATE, count=1, device_id=2)
            self.assertEqual(response.registers, [3])
            client.close()

        await asyncio.to_thread(initial_operations)
        await asyncio.sleep(0.08)
        self.assertEqual(self.state.baud_rate, 19200)

        def verify_new_baud() -> None:
            client = self._client(baudrate=19200)
            response = client.read_holding_registers(BAUD_RATE, count=1, device_id=2)
            self.assertFalse(response.isError())
            self.assertEqual(response.registers, [3])
            client.close()

        await asyncio.to_thread(verify_new_baud)


if __name__ == "__main__":
    unittest.main()
