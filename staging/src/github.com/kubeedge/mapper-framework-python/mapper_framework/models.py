from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from .anycodec import decode_customized_value
from .generated import api_pb2


def resource_id(namespace: str, name: str) -> str:
    return f"{namespace}/{name}"


@dataclass(frozen=True)
class PropertyContext:
    name: str
    data_type: str = "string"
    access_mode: str = "ReadWrite"
    desired: str = ""
    desired_present: bool = False
    visitor_protocol: str = ""
    visitor_config: dict[str, Any] = field(default_factory=dict)
    collect_cycle_ms: int = 1000
    report_cycle_ms: int = 1000
    report_to_cloud: bool = False

    @classmethod
    def from_proto(cls, prop: api_pb2.DeviceProperty, model_property: Any = None) -> "PropertyContext":
        # KubeEdge's DeviceProperty.Desired is a value struct. During the
        # CRD-to-DMI conversion its zero value is serialized as ``desired: {}``,
        # which creates an empty protobuf message even when no desired value was
        # configured. Treat only a populated value or metadata as an actual
        # desired update so drivers do not try to write an empty string.
        desired_present = prop.HasField("desired") and bool(
            prop.desired.value or prop.desired.metadata
        )
        data_type = getattr(model_property, "type", "string") or "string"
        access_mode = getattr(model_property, "accessMode", "ReadWrite") or "ReadWrite"
        return cls(
            name=prop.name,
            data_type=data_type,
            access_mode=access_mode,
            desired=prop.desired.value if desired_present else "",
            desired_present=desired_present,
            visitor_protocol=prop.visitors.protocolName,
            visitor_config=decode_customized_value(prop.visitors.configData),
            collect_cycle_ms=prop.collectCycle or 1000,
            report_cycle_ms=prop.reportCycle or 1000,
            report_to_cloud=prop.reportToCloud,
        )


@dataclass
class DeviceContext:
    raw: api_pb2.Device
    model: api_pb2.DeviceModel | None
    id: str
    protocol_name: str
    protocol_config: dict[str, Any]
    properties: dict[str, PropertyContext]
    methods: dict[str, tuple[str, ...]]
    model_properties: dict[str, Any]

    @property
    def name(self) -> str:
        return self.raw.name

    @property
    def namespace(self) -> str:
        return self.raw.namespace

    @classmethod
    def from_proto(
        cls,
        device: api_pb2.Device,
        model: api_pb2.DeviceModel | None,
    ) -> "DeviceContext":
        model_properties: dict[str, Any] = {}
        if model is not None and model.HasField("spec"):
            model_properties = {prop.name: prop for prop in model.spec.properties}

        properties = {
            prop.name: PropertyContext.from_proto(prop, model_properties.get(prop.name))
            for prop in device.spec.properties
        }
        methods = {
            method.name: tuple(method.propertyNames)
            for method in device.spec.methods
        }
        return cls(
            raw=device,
            model=model,
            id=resource_id(device.namespace, device.name),
            protocol_name=device.spec.protocol.protocolName,
            protocol_config=decode_customized_value(device.spec.protocol.configData),
            properties=properties,
            methods=methods,
            model_properties=model_properties,
        )
