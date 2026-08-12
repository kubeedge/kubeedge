from __future__ import annotations

import json
from typing import Any

from google.protobuf import any_pb2, json_format, wrappers_pb2
from google.protobuf.message import Message


_WRAPPER_TYPES: tuple[type[Message], ...] = (
    wrappers_pb2.StringValue,
    wrappers_pb2.BoolValue,
    wrappers_pb2.Int32Value,
    wrappers_pb2.Int64Value,
    wrappers_pb2.UInt32Value,
    wrappers_pb2.UInt64Value,
    wrappers_pb2.FloatValue,
    wrappers_pb2.DoubleValue,
    wrappers_pb2.BytesValue,
)


def decode_any(value: any_pb2.Any) -> Any:
    """Decode the standard Any values used by KubeEdge CustomizedValue.

    Custom protobuf messages are decoded through Any's registered descriptor
    pool. An unknown type is returned as a diagnostic dictionary instead of
    crashing the mapper callback.
    """
    for message_type in _WRAPPER_TYPES:
        if value.Is(message_type.DESCRIPTOR):
            message = message_type()
            value.Unpack(message)
            return message.value

    try:
        message = value.Unpack(value._concrete_class()) if hasattr(value, "_concrete_class") else None
    except Exception:
        message = None
    if message is not None:
        return message

    try:
        message = value.UnpackNew()
        return json_format.MessageToDict(message, preserving_proto_field_name=True)
    except Exception:
        return {
            "@type": value.type_url,
            "value_base64": value.value.hex(),
        }


def decode_customized_value(value: Any) -> dict[str, Any]:
    if value is None:
        return {}
    return {key: decode_any(any_value) for key, any_value in value.data.items()}


def to_string(value: Any) -> str:
    """Match mapper-framework's string conversion for reported values."""
    if value is None:
        return ""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, bytes):
        return value.decode("utf-8", errors="replace")
    if isinstance(value, str):
        return value
    if isinstance(value, (int, float)):
        return str(value)
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"), default=str)

