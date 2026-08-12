from __future__ import annotations

import json
import threading
from datetime import datetime, timezone
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import unquote, urlparse

from .panel import DevicePanel


class _Handler(BaseHTTPRequestHandler):
    panel: DevicePanel

    def do_GET(self) -> None:  # noqa: N802
        try:
            self._dispatch()
        except KeyError as exc:
            self._send_error(HTTPStatus.NOT_FOUND, str(exc))
        except Exception as exc:
            self._send_error(HTTPStatus.INTERNAL_SERVER_ERROR, str(exc))

    def log_message(self, format: str, *args: object) -> None:
        return

    def _dispatch(self) -> None:
        path = [unquote(item) for item in urlparse(self.path).path.split("/") if item]
        if path == ["api", "v1", "ping"]:
            self._send_json(
                HTTPStatus.OK,
                {"message": "This is v1 API, the server is running normally."},
            )
            return

        if len(path) == 6 and path[:3] == ["api", "v1", "device"]:
            namespace, name, property_name = path[3:6]
            value, data_type = self.panel.read_property(namespace, name, property_name)
            self._send_json(
                HTTPStatus.OK,
                {
                    "data": {
                        "deviceName": name,
                        "propertyName": property_name,
                        "namespace": namespace,
                        "value": value,
                        "type": data_type,
                    }
                },
            )
            return

        if len(path) == 5 and path[:3] == ["api", "v1", "devicemethod"]:
            namespace, name = path[3:5]
            context, methods = self.panel.get_methods(namespace, name)
            self._send_json(
                HTTPStatus.OK,
                {
                    "data": {
                        "methods": [
                            {
                                "name": method,
                                "path": f"/api/v1/devicemethod/{namespace}/{name}/{method}/{{property}}/{{data}}",
                                "parameters": [
                                    {"propertyName": prop, "valueType": context.properties[prop].data_type}
                                    for prop in properties
                                ],
                            }
                            for method, properties in methods.items()
                        ]
                    }
                },
            )
            return

        if len(path) == 8 and path[:3] == ["api", "v1", "devicemethod"]:
            namespace, name, method, property_name, value = path[3:8]
            self.panel.write_property(namespace, name, method, property_name, value)
            self._send_json(HTTPStatus.OK, {"message": f"Write data {value} successfully."})
            return

        if len(path) == 6 and path[:4] == ["api", "v1", "meta", "model"]:
            namespace, name = path[4:6]
            context, _ = self.panel.get_methods(namespace, name)
            model = context.model
            self._send_json(HTTPStatus.OK, {"data": model})
            return

        if len(path) == 5 and path[:3] == ["api", "v1", "database"]:
            self._send_error(HTTPStatus.SERVICE_UNAVAILABLE, "database is not enabled")
            return

        self._send_error(HTTPStatus.NOT_FOUND, "route not found")

    def _send_json(self, status: HTTPStatus, payload: object) -> None:
        from google.protobuf.json_format import MessageToDict
        from google.protobuf.message import Message

        def normalize(value: object) -> object:
            if isinstance(value, Message):
                return MessageToDict(value, preserving_proto_field_name=True)
            if isinstance(value, dict):
                return {key: normalize(item) for key, item in value.items()}
            if isinstance(value, (list, tuple)):
                return [normalize(item) for item in value]
            return value

        payload = normalize(payload)
        body = {
            "apiVersion": "v1",
            "statusCode": int(status),
            "timeStamp": datetime.now(timezone.utc).isoformat(),
            **(payload if isinstance(payload, dict) else {"data": payload}),
        }
        encoded = json.dumps(body, ensure_ascii=False, default=str).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def _send_error(self, status: HTTPStatus, message: str) -> None:
        self._send_json(status, {"error": message})


class HTTPServer:
    def __init__(self, panel: DevicePanel, host: str, port: int) -> None:
        handler = type("MapperHTTPHandler", (_Handler,), {"panel": panel})
        self.server = ThreadingHTTPServer((host, port), handler)
        self.thread = threading.Thread(target=self.server.serve_forever, name="mapper-http", daemon=True)

    def start(self) -> None:
        self.thread.start()

    def stop(self) -> None:
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)
