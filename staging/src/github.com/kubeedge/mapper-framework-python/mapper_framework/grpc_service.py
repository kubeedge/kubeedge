from __future__ import annotations

import grpc

from .generated import api_pb2, api_pb2_grpc
from .panel import DevicePanel


class DeviceMapperService(api_pb2_grpc.DeviceMapperServiceServicer):
    """DMI server implemented by the framework; users only implement drivers."""

    def __init__(self, panel: DevicePanel) -> None:
        self.panel = panel

    def RegisterDevice(self, request, context):
        device = request.device
        if not request.HasField("device") or not device.name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "device is required")
        try:
            self.panel.register_device(device)
        except Exception as exc:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, str(exc))
        return api_pb2.RegisterDeviceResponse(
            deviceName=device.name,
            deviceNamespace=device.namespace,
        )

    def RemoveDevice(self, request, context):
        if not request.deviceName:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "deviceName is required")
        try:
            self.panel.remove_device(request.deviceNamespace, request.deviceName)
        except Exception as exc:
            context.abort(grpc.StatusCode.INTERNAL, str(exc))
        return api_pb2.RemoveDeviceResponse()

    def UpdateDevice(self, request, context):
        if not request.HasField("device") or not request.device.name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "device is required")
        try:
            self.panel.update_device(request.device)
        except Exception as exc:
            context.abort(grpc.StatusCode.FAILED_PRECONDITION, str(exc))
        return api_pb2.UpdateDeviceResponse()

    def CreateDeviceModel(self, request, context):
        if not request.HasField("model") or not request.model.name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "model is required")
        self.panel.create_model(request.model)
        return api_pb2.CreateDeviceModelResponse(
            deviceModelName=request.model.name,
            deviceModelNamespace=request.model.namespace,
        )

    def RemoveDeviceModel(self, request, context):
        if not request.modelName:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "modelName is required")
        self.panel.remove_model(request.modelNamespace, request.modelName)
        return api_pb2.RemoveDeviceModelResponse()

    def UpdateDeviceModel(self, request, context):
        if not request.HasField("model") or not request.model.name:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "model is required")
        self.panel.update_model(request.model)
        return api_pb2.UpdateDeviceModelResponse()

    def GetDevice(self, request, context):
        if not request.deviceName:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "deviceName is required")
        try:
            device = self.panel.get_device(request.deviceNamespace, request.deviceName)
        except KeyError as exc:
            context.abort(grpc.StatusCode.NOT_FOUND, str(exc))
        return api_pb2.GetDeviceResponse(device=device)

