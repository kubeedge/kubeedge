from __future__ import annotations

import json
from datetime import datetime, timezone
from typing import Any

from google.protobuf import json_format, wrappers_pb2  # noqa: F401
from google.protobuf.message import Message


def emit(event: str, payload: Any) -> None:
    payload = _json_value(payload)
    timestamp = datetime.now(timezone.utc).isoformat(timespec="milliseconds")
    print(
        f"\n[{timestamp}] {event}\n"
        f"{json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True, default=str)}",
        flush=True,
    )


def _json_value(value: Any) -> Any:
    if isinstance(value, Message):
        try:
            return json_format.MessageToDict(value, preserving_proto_field_name=True)
        except (TypeError, ValueError):
            return {"protobuf": str(value)}
    if isinstance(value, dict):
        return {key: _json_value(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [_json_value(item) for item in value]
    return value
