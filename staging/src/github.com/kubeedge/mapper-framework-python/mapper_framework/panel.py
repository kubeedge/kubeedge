from __future__ import annotations

import logging
import threading
from dataclasses import dataclass, field
from typing import Callable, Any

import grpc

from .anycodec import to_string
from .driver import BaseDeviceDriver, DriverRegistry
from .events import emit
from .generated import api_pb2
from .models import DeviceContext, PropertyContext, resource_id


LOGGER = logging.getLogger("kubeedge.python-mapper-framework")
StatusReporter = Callable[[api_pb2.ReportDeviceStatusRequest], None]
StateReporter = Callable[[api_pb2.ReportDeviceStatesRequest], None]


def convert_value(value: str, data_type: str) -> Any:
    kind = (data_type or "string").strip().lower()
    if kind in {"int", "integer", "int32", "int64"}:
        return int(value)
    if kind in {"float", "float32"}:
        return float(value)
    if kind in {"double", "float64"}:
        return float(value)
    if kind in {"bool", "boolean"}:
        return value.strip().lower() in {"true", "1", "yes", "on"}
    return value


@dataclass
class _Session:
    context: DeviceContext
    driver: BaseDeviceDriver
    stop_event: threading.Event = field(default_factory=threading.Event)
    threads: list[threading.Thread] = field(default_factory=list)
    reported: dict[str, str] = field(default_factory=dict)

    def start(self, panel: "DevicePanel") -> None:
        for property_context in self.context.properties.values():
            if property_context.report_to_cloud:
                thread = threading.Thread(
                    target=panel._collect_loop,
                    args=(self, property_context),
                    name=f"collect-{self.context.id}-{property_context.name}",
                    daemon=True,
                )
                self.threads.append(thread)
                thread.start()

        if self.context.raw.status.reportToCloud:
            thread = threading.Thread(
                target=panel._state_loop,
                args=(self,),
                name=f"state-{self.context.id}",
                daemon=True,
            )
            self.threads.append(thread)
            thread.start()

    def stop(self) -> None:
        self.stop_event.set()
        for thread in self.threads:
            thread.join(timeout=2)
        self.threads.clear()
        self.driver.stop_device()


class DevicePanel:
    """Device lifecycle, driver execution, polling and DMI reporting."""

    def __init__(
        self,
        registry: DriverRegistry,
        report_status: StatusReporter,
        report_states: StateReporter,
        reconnect_interval_seconds: float = 5.0,
    ) -> None:
        self.registry = registry
        self.report_status = report_status
        self.report_states = report_states
        self.reconnect_interval_seconds = reconnect_interval_seconds
        self.models: dict[str, api_pb2.DeviceModel] = {}
        self.devices: dict[str, DeviceContext] = {}
        self.sessions: dict[str, _Session] = {}
        self.pending: dict[str, api_pb2.Device] = {}
        self._lock = threading.RLock()
        self._last_reported: dict[str, dict[str, str]] = {}
        self._stopped = threading.Event()
        self._retry_thread = threading.Thread(target=self._retry_loop, name="device-retry", daemon=True)
        self._retry_thread.start()

    def initialize(
        self,
        devices: list[api_pb2.Device],
        models: list[api_pb2.DeviceModel],
    ) -> None:
        for model in models:
            self.update_model(model)
        for device in devices:
            try:
                self.register_device(device)
            except Exception as exc:
                self._mark_pending(device, exc)

    def create_model(self, model: api_pb2.DeviceModel) -> None:
        self.update_model(model)
        emit("CREATE_DEVICE_MODEL", model)

    def update_model(self, model: api_pb2.DeviceModel) -> None:
        copy = api_pb2.DeviceModel()
        copy.CopyFrom(model)
        with self._lock:
            self.models[resource_id(model.namespace, model.name)] = copy
        emit("UPDATE_DEVICE_MODEL", model)

    def remove_model(self, namespace: str, name: str) -> None:
        with self._lock:
            self.models.pop(resource_id(namespace, name), None)
        emit("REMOVE_DEVICE_MODEL", {"model_id": resource_id(namespace, name)})

    def register_device(self, device: api_pb2.Device) -> None:
        self._start_device(device, event="REGISTER_DEVICE")

    def update_device(self, device: api_pb2.Device) -> None:
        self._start_device(device, event="UPDATE_DEVICE")

    def remove_device(self, namespace: str, name: str) -> None:
        device_id = resource_id(namespace, name)
        with self._lock:
            session = self.sessions.pop(device_id, None)
            self.devices.pop(device_id, None)
            self.pending.pop(device_id, None)
            self._last_reported.pop(device_id, None)
        if session is not None:
            session.stop()
        emit("REMOVE_DEVICE", {"device_id": device_id})

    def get_device(self, namespace: str, name: str) -> api_pb2.Device:
        device_id = resource_id(namespace, name)
        with self._lock:
            context = self.devices.get(device_id)
            if context is None:
                raise KeyError(device_id)
            result = api_pb2.Device()
            result.CopyFrom(context.raw)
            reported = self._last_reported.get(device_id, {})
        result.status.twins.clear()
        for property_context in context.properties.values():
            if property_context.name not in reported:
                continue
            result.status.twins.add(
                propertyName=property_context.name,
                observedDesired=api_pb2.TwinProperty(value=property_context.desired),
                reported=api_pb2.TwinProperty(value=reported[property_context.name]),
            )
        return result

    def get_model(self, namespace: str, name: str) -> api_pb2.DeviceModel:
        with self._lock:
            model = self.models.get(resource_id(namespace, name))
            if model is None:
                raise KeyError(resource_id(namespace, name))
            result = api_pb2.DeviceModel()
            result.CopyFrom(model)
            return result

    def read_property(self, namespace: str, name: str, property_name: str) -> tuple[str, str]:
        context, session = self._get_session(namespace, name)
        prop = context.properties.get(property_name)
        if prop is None:
            raise KeyError(f"property {property_name!r} not found")
        value = to_string(session.driver.get_device_data(prop))
        with self._lock:
            self._last_reported.setdefault(context.id, {})[property_name] = value
        return value, prop.data_type

    def write_property(
        self,
        namespace: str,
        name: str,
        method_name: str,
        property_name: str,
        value: str,
    ) -> None:
        context, session = self._get_session(namespace, name)
        prop = context.properties.get(property_name)
        if prop is None:
            raise KeyError(f"property {property_name!r} not found")
        session.driver.device_data_write(method_name, prop, convert_value(value, prop.data_type))

    def get_methods(self, namespace: str, name: str) -> tuple[DeviceContext, dict[str, tuple[str, ...]]]:
        context, _ = self._get_session(namespace, name)
        return context, context.methods

    def stop(self) -> None:
        self._stopped.set()
        with self._lock:
            sessions = list(self.sessions.values())
            self.sessions.clear()
            self.devices.clear()
        for session in sessions:
            session.stop()

    def _start_device(self, device: api_pb2.Device, event: str) -> None:
        if not device.HasField("spec"):
            raise ValueError(f"device {device.name!r} has no spec")
        device_id = resource_id(device.namespace, device.name)
        model_id = resource_id(device.namespace, device.spec.deviceModelReference)
        with self._lock:
            model = self.models.get(model_id)
        if model is None:
            raise KeyError(f"device model {model_id!r} not found")

        old_session = None
        with self._lock:
            old_session = self.sessions.pop(device_id, None)
            self.devices.pop(device_id, None)
        if old_session is not None:
            old_session.stop()

        context = DeviceContext.from_proto(device, model)
        driver = self.registry.create(context)
        try:
            driver.init_device()
            session = _Session(context=context, driver=driver)
            with self._lock:
                self.devices[device_id] = context
                self.sessions[device_id] = session
                self.pending.pop(device_id, None)
                self._last_reported.setdefault(device_id, {})
            self._apply_desired(session)
            session.start(self)
            emit(event, device)
        except Exception:
            try:
                driver.stop_device()
            finally:
                with self._lock:
                    self.devices.pop(device_id, None)
                    self.sessions.pop(device_id, None)
                raise

    def _apply_desired(self, session: _Session) -> None:
        context = session.context
        for property_context in context.properties.values():
            if not property_context.desired_present:
                continue
            emit(
                "CLOUD_DESIRED_VALUE",
                {
                    "device_id": context.id,
                    "property": property_context.name,
                    "value": property_context.desired,
                },
            )
            if property_context.access_mode.lower() == "readonly":
                emit(
                    "DESIRED_VALUE_SKIPPED",
                    {"device_id": context.id, "property": property_context.name, "reason": "ReadOnly"},
                )
                continue
            value = convert_value(property_context.desired, property_context.data_type)
            session.driver.set_device_data(value, property_context)

    def _collect_loop(self, session: _Session, property_context: PropertyContext) -> None:
        interval = max(property_context.collect_cycle_ms, 1) / 1000
        while not session.stop_event.wait(interval):
            try:
                value = to_string(session.driver.get_device_data(property_context))
                with self._lock:
                    session.reported[property_context.name] = value
                    self._last_reported.setdefault(session.context.id, {})[property_context.name] = value
                self.report_status(
                    api_pb2.ReportDeviceStatusRequest(
                        deviceName=session.context.name,
                        deviceNamespace=session.context.namespace,
                        reportedDevice=api_pb2.DeviceStatus(
                            twins=[
                                api_pb2.Twin(
                                    propertyName=property_context.name,
                                    observedDesired=api_pb2.TwinProperty(value=property_context.desired),
                                    reported=api_pb2.TwinProperty(
                                        value=value,
                                        metadata={"type": property_context.data_type},
                                    ),
                                )
                            ]
                        ),
                    )
                )
            except Exception as exc:
                emit(
                    "DEVICE_PROPERTY_READ_FAILED",
                    {"device_id": session.context.id, "property": property_context.name, "error": str(exc)},
                )

    def _state_loop(self, session: _Session) -> None:
        interval = max(session.context.raw.status.reportCycle or 1000, 1) / 1000
        while not session.stop_event.wait(interval):
            try:
                state = to_string(session.driver.get_device_states())
                self.report_states(
                    api_pb2.ReportDeviceStatesRequest(
                        deviceName=session.context.name,
                        deviceNamespace=session.context.namespace,
                        state=state,
                    )
                )
            except Exception as exc:
                emit("DEVICE_STATE_READ_FAILED", {"device_id": session.context.id, "error": str(exc)})

    def _get_session(self, namespace: str, name: str) -> tuple[DeviceContext, _Session]:
        device_id = resource_id(namespace, name)
        with self._lock:
            context = self.devices.get(device_id)
            session = self.sessions.get(device_id)
        if context is None or session is None:
            raise KeyError(device_id)
        return context, session

    def _mark_pending(self, device: api_pb2.Device, error: Exception) -> None:
        with self._lock:
            self.pending[resource_id(device.namespace, device.name)] = device
        emit(
            "DEVICE_START_PENDING",
            {"device_id": resource_id(device.namespace, device.name), "error": str(error)},
        )

    def _retry_loop(self) -> None:
        while not self._stopped.wait(self.reconnect_interval_seconds):
            with self._lock:
                pending = list(self.pending.values())
            for device in pending:
                try:
                    self._start_device(device, event="DEVICE_RECONNECTED")
                except Exception as exc:
                    emit(
                        "DEVICE_RECONNECT_FAILED",
                        {"device_id": resource_id(device.namespace, device.name), "error": str(exc)},
                    )

