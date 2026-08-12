from __future__ import annotations

import tempfile
import unittest
from concurrent import futures
from pathlib import Path

import grpc

from mapper_framework.config import Config, CommonConfig, GRPCServerConfig, HTTPServerConfig
from mapper_framework.generated import api_pb2, api_pb2_grpc
from mapper_framework.runtime import MapperRuntime


class FakeEdgeCore(api_pb2_grpc.DeviceManagerServiceServicer):
    def __init__(self, device, model):
        self.device = device
        self.model = model

    def MapperRegister(self, request, context):
        return api_pb2.MapperRegisterResponse(deviceList=[self.device], modelList=[self.model])


class RuntimeSmokeTest(unittest.TestCase):
    def test_runtime_registers_and_initializes_print_driver(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            edge_socket = root / "dmi.sock"
            mapper_socket = root / "mapper.sock"

            model = api_pb2.DeviceModel(name="model", namespace="default")
            model.spec.properties.add(name="message", type="STRING", accessMode="ReadWrite")
            device = api_pb2.Device(name="device", namespace="default")
            device.spec.deviceModelReference = "model"
            device.spec.protocol.protocolName = "python-demo"
            device.spec.properties.add(name="message").desired.value = "hello"

            edge_server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
            api_pb2_grpc.add_DeviceManagerServiceServicer_to_server(
                FakeEdgeCore(device, model), edge_server
            )
            self.assertNotEqual(edge_server.add_insecure_port(f"unix://{edge_socket}"), 0)
            edge_server.start()

            config = Config(
                grpc_server=GRPCServerConfig(str(mapper_socket)),
                common=CommonConfig(
                    name="test-mapper",
                    version="v0.1.0",
                    api_version="v1beta1",
                    protocol="python-demo",
                    address="127.0.0.1",
                    edgecore_sock=str(edge_socket),
                    reconnect_interval_seconds=60,
                ),
                http_server=HTTPServerConfig(enabled=False),
            )
            runtime = MapperRuntime(config)
            try:
                runtime.start()
                self.assertIn("default/device", runtime.panel.devices)
                self.assertTrue(mapper_socket.exists())
                channel = grpc.insecure_channel(f"unix://{mapper_socket}")
                grpc.channel_ready_future(channel).result(timeout=2)
                stub = api_pb2_grpc.DeviceMapperServiceStub(channel)
                updated = api_pb2.Device()
                updated.CopyFrom(device)
                updated.spec.properties[0].desired.value = "updated"
                stub.UpdateDevice(api_pb2.UpdateDeviceRequest(device=updated), timeout=2)
                current = stub.GetDevice(
                    api_pb2.GetDeviceRequest(deviceName="device", deviceNamespace="default"),
                    timeout=2,
                )
                self.assertEqual(current.device.spec.properties[0].desired.value, "updated")
                channel.close()
            finally:
                runtime.stop()
                edge_server.stop(0)
            self.assertFalse(mapper_socket.exists())


if __name__ == "__main__":
    unittest.main()
