from __future__ import annotations

import importlib
import os
import signal
from concurrent import futures
from typing import Iterable

import grpc

from .config import Config
from .driver import BaseDeviceDriver, DriverRegistry, PrintDriver
from .edge_client import EdgeCoreClient
from .events import emit
from .generated import api_pb2_grpc
from .grpc_service import DeviceMapperService
from .http_server import HTTPServer
from .panel import DevicePanel


class MapperRuntime:
    def __init__(self, config: Config, driver_types: Iterable[type[BaseDeviceDriver]] = ()) -> None:
        self.config = config
        self.edge_client = EdgeCoreClient(config)
        self.registry = DriverRegistry()
        self.registry.register(PrintDriver)
        for driver_type in driver_types:
            self.registry.register(driver_type)
        self.panel = DevicePanel(
            registry=self.registry,
            report_status=self.edge_client.report_device_status,
            report_states=self.edge_client.report_device_states,
            reconnect_interval_seconds=config.common.reconnect_interval_seconds,
        )
        self.grpc_server = grpc.server(futures.ThreadPoolExecutor(max_workers=32))
        api_pb2_grpc.add_DeviceMapperServiceServicer_to_server(
            DeviceMapperService(self.panel), self.grpc_server
        )
        self.http_server: HTTPServer | None = None
        self._stopped = False

    def start(self) -> None:
        socket_path = self.config.grpc_server.socket_path
        parent = os.path.dirname(socket_path)
        if parent:
            os.makedirs(parent, exist_ok=True)
        if os.path.exists(socket_path):
            os.remove(socket_path)
        if self.grpc_server.add_insecure_port(f"unix://{socket_path}") == 0:
            raise RuntimeError(f"failed to bind mapper Unix Socket: {socket_path}")
        self.grpc_server.start()
        emit(
            "MAPPER_GRPC_SERVER_STARTED",
            {"socket_path": socket_path, "protocol": self.config.common.protocol},
        )

        if self.config.http_server.enabled:
            self.http_server = HTTPServer(
                self.panel,
                self.config.http_server.host,
                self.config.http_server.port,
            )
            self.http_server.start()
            emit(
                "MAPPER_HTTP_SERVER_STARTED",
                {"host": self.config.http_server.host, "port": self.config.http_server.port},
            )

        self.edge_client.connect()
        response = self.edge_client.register()
        self.panel.initialize(list(response.deviceList), list(response.modelList))

    def stop(self, *_args: object) -> None:
        if self._stopped:
            return
        self._stopped = True
        emit("MAPPER_STOPPING", {})
        self.panel.stop()
        if self.http_server is not None:
            self.http_server.stop()
        self.edge_client.close()
        self.grpc_server.stop(grace=2)
        socket_path = self.config.grpc_server.socket_path
        if os.path.exists(socket_path):
            os.remove(socket_path)

    def wait(self) -> None:
        try:
            self.grpc_server.wait_for_termination()
        finally:
            self.stop()


def load_driver_type(spec: str) -> type[BaseDeviceDriver]:
    try:
        module_name, class_name = spec.split(":", 1)
    except ValueError as exc:
        raise ValueError(f"driver must use MODULE:CLASS format, got {spec!r}") from exc
    module = importlib.import_module(module_name)
    driver_type = getattr(module, class_name)
    if not isinstance(driver_type, type) or not issubclass(driver_type, BaseDeviceDriver):
        raise TypeError(f"{spec} is not a BaseDeviceDriver subclass")
    return driver_type


def install_signal_handlers(runtime: MapperRuntime) -> None:
    signal.signal(signal.SIGINT, runtime.stop)
    signal.signal(signal.SIGTERM, runtime.stop)

