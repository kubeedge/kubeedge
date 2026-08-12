from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any, Mapping

import yaml


@dataclass(frozen=True)
class GRPCServerConfig:
    socket_path: str


@dataclass(frozen=True)
class HTTPServerConfig:
    enabled: bool = True
    host: str = "0.0.0.0"
    port: int = 7777


@dataclass(frozen=True)
class CommonConfig:
    name: str
    version: str
    api_version: str
    protocol: str
    address: str
    edgecore_sock: str
    register_with_data: bool = True
    request_timeout_seconds: float = 10.0
    reconnect_interval_seconds: float = 5.0


@dataclass(frozen=True)
class Config:
    grpc_server: GRPCServerConfig
    common: CommonConfig
    http_server: HTTPServerConfig

    @classmethod
    def load(cls, path: str | Path) -> "Config":
        with Path(path).open("r", encoding="utf-8") as stream:
            raw = yaml.safe_load(stream) or {}

        grpc_raw = _mapping(raw, "grpc_server")
        common_raw = _mapping(raw, "common")
        http_raw = raw.get("http_server", {})
        if not isinstance(http_raw, Mapping):
            raise ValueError("configuration section 'http_server' must be a mapping")

        return cls(
            grpc_server=GRPCServerConfig(socket_path=_required_string(grpc_raw, "socket_path")),
            common=CommonConfig(
                name=_required_string(common_raw, "name"),
                version=_required_string(common_raw, "version"),
                api_version=_required_string(common_raw, "api_version"),
                protocol=_required_string(common_raw, "protocol"),
                address=str(common_raw.get("address", "127.0.0.1")),
                edgecore_sock=_required_string(common_raw, "edgecore_sock"),
                register_with_data=bool(common_raw.get("register_with_data", True)),
                request_timeout_seconds=float(common_raw.get("request_timeout_seconds", 10)),
                reconnect_interval_seconds=float(common_raw.get("reconnect_interval_seconds", 5)),
            ),
            http_server=HTTPServerConfig(
                enabled=bool(http_raw.get("enabled", True)),
                host=str(http_raw.get("host", "0.0.0.0")),
                port=int(http_raw.get("port", 7777)),
            ),
        )


def _mapping(raw: Mapping[str, Any], key: str) -> Mapping[str, Any]:
    value = raw.get(key)
    if not isinstance(value, Mapping):
        raise ValueError(f"configuration section {key!r} must be a mapping")
    return value


def _required_string(raw: Mapping[str, Any], key: str) -> str:
    value = raw.get(key)
    if value is None or not str(value).strip():
        raise ValueError(f"configuration field {key!r} is required")
    return str(value)

