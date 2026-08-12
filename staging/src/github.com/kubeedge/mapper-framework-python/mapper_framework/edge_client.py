from __future__ import annotations

import grpc

from .config import Config
from .events import emit
from .generated import api_pb2, api_pb2_grpc


def unix_target(socket_path: str) -> str:
    if socket_path.startswith("unix://"):
        return socket_path
    return f"unix://{socket_path}" if socket_path.startswith("/") else f"unix:///{socket_path}"


class EdgeCoreClient:
    """Persistent DMI client for EdgeCore's DeviceManagerService."""

    def __init__(self, config: Config) -> None:
        self.config = config
        self.channel: grpc.Channel | None = None
        self.stub: api_pb2_grpc.DeviceManagerServiceStub | None = None

    def connect(self) -> None:
        self.channel = grpc.insecure_channel(unix_target(self.config.common.edgecore_sock))
        grpc.channel_ready_future(self.channel).result(
            timeout=self.config.common.request_timeout_seconds
        )
        self.stub = api_pb2_grpc.DeviceManagerServiceStub(self.channel)

    def register(self) -> api_pb2.MapperRegisterResponse:
        if self.stub is None:
            raise RuntimeError("EdgeCore client is not connected")
        request = api_pb2.MapperRegisterRequest(
            withData=self.config.common.register_with_data,
            mapper=api_pb2.MapperInfo(
                name=self.config.common.name,
                version=self.config.common.version,
                api_version=self.config.common.api_version,
                protocol=self.config.common.protocol,
                address=self.config.grpc_server.socket_path.encode("utf-8"),
                state="ok",
            ),
        )
        emit("MAPPER_REGISTER_REQUEST", request_to_dict(request))
        response = self.stub.MapperRegister(
            request,
            timeout=self.config.common.request_timeout_seconds,
        )
        emit("MAPPER_REGISTER_RESPONSE", response_to_dict(response))
        return response

    def report_device_status(self, request: api_pb2.ReportDeviceStatusRequest) -> None:
        if self.stub is None:
            raise RuntimeError("EdgeCore client is not connected")
        emit("REPORT_DEVICE_STATUS_REQUEST", request_to_dict(request))
        self.stub.ReportDeviceStatus(
            request,
            timeout=self.config.common.request_timeout_seconds,
        )

    def report_device_states(self, request: api_pb2.ReportDeviceStatesRequest) -> None:
        if self.stub is None:
            raise RuntimeError("EdgeCore client is not connected")
        emit("REPORT_DEVICE_STATES_REQUEST", request_to_dict(request))
        self.stub.ReportDeviceStates(
            request,
            timeout=self.config.common.request_timeout_seconds,
        )

    def close(self) -> None:
        if self.channel is not None:
            self.channel.close()
        self.channel = None
        self.stub = None


def request_to_dict(message: object) -> object:
    from google.protobuf.json_format import MessageToDict
    from google.protobuf.message import Message

    if isinstance(message, Message):
        return MessageToDict(message, preserving_proto_field_name=True)
    return message


def response_to_dict(message: object) -> object:
    return request_to_dict(message)

