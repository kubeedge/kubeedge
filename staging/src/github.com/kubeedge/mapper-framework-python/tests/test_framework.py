from __future__ import annotations

import unittest
from urllib.request import urlopen

from google.protobuf.wrappers_pb2 import Int32Value, StringValue

from mapper_framework.anycodec import decode_customized_value, to_string
from mapper_framework.driver import BaseDeviceDriver, DriverRegistry
from mapper_framework.generated import api_pb2
from mapper_framework.models import DeviceContext
from mapper_framework.http_server import HTTPServer
from mapper_framework.panel import DevicePanel


class FakeDriver(BaseDeviceDriver):
    protocol_name = "fake"
    created = []
    writes = []

    def init_device(self):
        self.created.append(self.device.id)

    def get_device_data(self, property_context):
        return 42

    def set_device_data(self, value, property_context):
        self.writes.append((self.device.id, property_context.name, value))


def make_device():
    device = api_pb2.Device(name="device", namespace="default")
    device.spec.deviceModelReference = "model"
    device.spec.protocol.protocolName = "fake"
    device.spec.protocol.configData.data["address"].Pack(StringValue(value="127.0.0.1"))
    prop = device.spec.properties.add(name="count", reportToCloud=False)
    prop.desired.value = "7"
    return device


def make_model():
    model = api_pb2.DeviceModel(name="model", namespace="default")
    model.spec.properties.add(name="count", type="INT32", accessMode="ReadWrite")
    return model


class FrameworkTest(unittest.TestCase):
    def test_any_wrapper_and_conversion(self):
        device = make_device()
        device.spec.protocol.configData.data["count"].Pack(Int32Value(value=3))
        values = decode_customized_value(device.spec.protocol.configData)
        self.assertEqual(values["address"], "127.0.0.1")
        self.assertEqual(values["count"], 3)
        self.assertEqual(to_string(True), "true")

    def test_device_context(self):
        context = DeviceContext.from_proto(make_device(), make_model())
        self.assertEqual(context.id, "default/device")
        self.assertEqual(context.protocol_config["address"], "127.0.0.1")
        self.assertEqual(context.properties["count"].data_type, "INT32")

    def test_empty_desired_message_is_treated_as_absent(self):
        device = make_device()
        prop = device.spec.properties[0]
        prop.ClearField("desired")
        prop.desired.SetInParent()

        context = DeviceContext.from_proto(device, make_model())

        self.assertFalse(context.properties["count"].desired_present)

    def test_zero_desired_value_is_treated_as_present(self):
        device = make_device()
        device.spec.properties[0].desired.value = "0"

        context = DeviceContext.from_proto(device, make_model())

        self.assertTrue(context.properties["count"].desired_present)
        self.assertEqual(context.properties["count"].desired, "0")

    def test_panel_creates_driver_and_converts_desired(self):
        FakeDriver.created.clear()
        FakeDriver.writes.clear()
        registry = DriverRegistry()
        registry.register(FakeDriver)
        reports = []
        states = []
        panel = DevicePanel(registry, reports.append, states.append, reconnect_interval_seconds=60)
        try:
            panel.update_model(make_model())
            panel.register_device(make_device())
            self.assertEqual(FakeDriver.created, ["default/device"])
            self.assertEqual(FakeDriver.writes, [("default/device", "count", 7)])
            self.assertEqual(panel.read_property("default", "device", "count"), ("42", "INT32"))
            panel.remove_device("default", "device")
            self.assertEqual(panel.devices, {})
        finally:
            panel.stop()

    def test_http_ping(self):
        registry = DriverRegistry()
        registry.register(FakeDriver)
        panel = DevicePanel(registry, lambda request: None, lambda request: None, reconnect_interval_seconds=60)
        server = HTTPServer(panel, "127.0.0.1", 0)
        try:
            server.start()
            with urlopen(f"http://127.0.0.1:{server.server.server_port}/api/v1/ping", timeout=2) as response:
                self.assertEqual(response.status, 200)
        finally:
            server.stop()
            panel.stop()


if __name__ == "__main__":
    unittest.main()
