from __future__ import annotations

import logging
from abc import ABC
from typing import Any, Type

from .events import emit
from .models import DeviceContext, PropertyContext


LOGGER = logging.getLogger("kubeedge.python-mapper-framework")


class BaseDeviceDriver(ABC):
    """Base class that users extend for one real-device protocol.

    A new driver instance is created for every Device. The method names mirror
    the Go mapper-framework template so a device implementation only needs to
    provide hardware-specific logic.
    """

    protocol_name = ""

    def __init__(self, device: DeviceContext) -> None:
        self.device = device

    def init_device(self) -> None:
        """Open the hardware connection and validate the protocol config."""

    def get_device_data(self, property_context: PropertyContext) -> Any:
        """Read one property from the device."""
        raise NotImplementedError

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        """Write one property to the device."""
        raise NotImplementedError

    def device_data_write(
        self,
        method_name: str,
        property_context: PropertyContext,
        value: Any,
    ) -> None:
        """Handle an explicit method write, used by the optional HTTP API."""
        self.set_device_data(value, property_context)

    def get_device_states(self) -> str:
        """Return a JSON string or a short state string for state reporting."""
        return "ok"

    def stop_device(self) -> None:
        """Close the hardware connection and release all resources."""


class PrintDriver(BaseDeviceDriver):
    """Default no-hardware driver; prints every cloud desired value."""

    protocol_name = "python-demo"

    def init_device(self) -> None:
        emit("DEVICE_DRIVER_INITIALIZED", {"device_id": self.device.id, "driver": type(self).__name__})

    def get_device_data(self, property_context: PropertyContext) -> Any:
        return property_context.desired

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        emit(
            "CLOUD_VALUE_RECEIVED",
            {
                "device_id": self.device.id,
                "property": property_context.name,
                "value": value,
            },
        )

    def stop_device(self) -> None:
        emit("DEVICE_DRIVER_STOPPED", {"device_id": self.device.id})


DriverType = Type[BaseDeviceDriver]


class DriverRegistry:
    def __init__(self) -> None:
        self._drivers: dict[str, DriverType] = {}

    def register(self, driver_type: DriverType) -> None:
        protocol = str(getattr(driver_type, "protocol_name", "")).strip()
        if not protocol:
            raise ValueError(f"driver {driver_type.__name__} must define protocol_name")
        self._drivers[protocol] = driver_type

    def create(self, device: DeviceContext) -> BaseDeviceDriver:
        driver_type = self._drivers.get(device.protocol_name)
        if driver_type is None:
            raise ValueError(
                f"no driver registered for protocol {device.protocol_name!r}; "
                f"registered protocols: {sorted(self._drivers)}"
            )
        return driver_type(device)

