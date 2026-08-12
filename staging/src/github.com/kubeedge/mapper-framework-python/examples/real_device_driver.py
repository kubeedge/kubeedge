"""Minimal driver template for a real device.

Start it with:

    python -m mapper_framework --config config.yaml \
      --driver examples.real_device_driver:ExampleDeviceDriver

Change protocol_name to the protocol in the cloud Device resource and replace
the TODO bodies with serial/TCP/Modbus/vendor-SDK operations.
"""

from __future__ import annotations

from typing import Any

from mapper_framework import BaseDeviceDriver, DeviceContext, PropertyContext


class ExampleDeviceDriver(BaseDeviceDriver):
    protocol_name = "python-demo"

    def __init__(self, device: DeviceContext) -> None:
        super().__init__(device)
        self.connection = None

    def init_device(self) -> None:
        # Example: read self.device.protocol_config and open a serial/TCP client.
        # port = self.device.protocol_config["port"]
        # self.connection = serial.Serial(port=port, baudrate=9600, timeout=1)
        self.connection = object()

    def get_device_data(self, property_context: PropertyContext) -> Any:
        # Example: read and decode a register selected by visitor_config.
        raise NotImplementedError("replace with a real device read")

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        # Example: encode value and write a register selected by visitor_config.
        raise NotImplementedError("replace with a real device write")

    def device_data_write(
        self,
        method_name: str,
        property_context: PropertyContext,
        value: Any,
    ) -> None:
        if method_name not in self.device.methods:
            raise ValueError(f"unsupported method: {method_name}")
        if property_context.name not in self.device.methods[method_name]:
            raise ValueError(f"unsupported property: {property_context.name}")
        self.set_device_data(value, property_context)

    def get_device_states(self) -> str:
        return '{"connected": true}'

    def stop_device(self) -> None:
        self.connection = None
