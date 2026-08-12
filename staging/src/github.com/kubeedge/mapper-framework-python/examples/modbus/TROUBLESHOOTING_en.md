# Modbus RTU Mapper Troubleshooting Guide

This guide records the fault chain encountered while connecting the industrial
thermostat example to a Modbus simulator. Diagnose in order: cloud resources,
CloudCore, EdgeCore/DMI, Mapper, then serial/Modbus. This avoids misidentifying a
DMI delivery failure as a serial communication problem.

## 1. Identify the failing layer

| Symptom or log | Meaning | Next step |
| --- | --- | --- |
| `/api/v1/ping` returns 200 | Mapper HTTP server works | Does not prove Device registration |
| API returns 404 and `'default/industrial-thermostat'` | No active Device session | Check DMI delivery or `DEVICE_START_PENDING` |
| `MAPPER_REGISTER_RESPONSE {}` | EdgeCore DMI returned no Devices/Models | Check CloudCore delivery and DMI cache |
| Response has `deviceList`, `modelList` | Cloud-to-EdgeCore resource path works | Check device initialization |
| `DEVICE_START_PENDING` | Device was delivered but driver initialization failed | Read its `error` |
| `REGISTER_DEVICE` / `DEVICE_RECONNECTED` | Driver initialization succeeded | HTTP I/O APIs may be used |

The 404 below says `default/industrial-thermostat` is absent from the framework
`devices`/`sessions` tables; it is not a property-name error.

```json
{"statusCode": 404, "error": "'default/industrial-thermostat'"}
```

## 2. Check cloud resources first

```bash
kubectl get nodes -o wide
kubectl get devices.devices.kubeedge.io -A
kubectl get devices.devices.kubeedge.io industrial-thermostat -n default \
  -o jsonpath='{.spec.nodeName}{"\n"}{.spec.protocol.protocolName}{"\n"}{.spec.deviceModelRef.name}{"\n"}'
kubectl get devicemodels.devices.kubeedge.io industrial-thermostat-model -n default
```

Expected key values are the actual edge-node name, `modbus-rtu`, and
`industrial-thermostat-model`. `spec.nodeName` must exactly match the first column
of `kubectl get nodes`, `spec.protocol.protocolName` must match
`mapper-config.yaml` `common.protocol`, and Device/DeviceModel must share a
namespace.

## 3. CloudCore lacks DeviceStatus access

When resources and node settings are correct but Mapper registration is empty, run:

```bash
kubectl get crd devicestatuses.devices.kubeedge.io
kubectl get devicestatuses.devices.kubeedge.io industrial-thermostat -n default
kubectl -n kubeedge logs deployment/cloudcore --since=30m |
  grep -Ei 'DeviceStatus|forbidden|devicecontroller|failed|error'
```

`devices/status` and `devicestatuses` are different Kubernetes resources. If the
CloudCore ServiceAccount cannot list `devicestatuses`, its informer cannot create
DeviceStatus. Grant the minimally required RBAC (adjust the ServiceAccount for your
installation):

```bash
kubectl apply -f - <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cloudcore-devicestatus
rules:
  - apiGroups: ["devices.kubeedge.io"]
    resources: ["devicestatuses", "devicestatuses/status"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cloudcore-devicestatus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cloudcore-devicestatus
subjects:
  - kind: ServiceAccount
    name: cloudcore
    namespace: kubeedge
EOF
kubectl auth can-i list devicestatuses.devices.kubeedge.io --as=system:serviceaccount:kubeedge:cloudcore
kubectl -n kubeedge rollout restart deployment/cloudcore
kubectl -n kubeedge rollout status deployment/cloudcore
```

The `get`, `list`, `watch`, `create`, `update`, `patch`, and `delete` operations
should be authorized as appropriate. Empty `spec: {}` and `status: {}` are normal
until a Mapper reports. Metrics tunnel errors and missing ImagePrePullJob CRDs are
separate from this HTTP 404, although they can indicate incomplete installation.

## 4. EdgeCore SQLite contains resources but DMI registration is empty

On the edge node:

```bash
sqlite3 /var/lib/kubeedge/edgecore.db \
  "SELECT key,type FROM meta WHERE value LIKE '%industrial-thermostat%';"
```

Expected entries include `default/devicemodel/industrial-thermostat-model` and
`default/device/industrial-thermostat`. If they exist but the register response is
still `{}`, EdgeCore's DMI in-memory cache may not contain the resources. Stop the
Mapper, restart EdgeCore, wait for `/etc/kubeedge/dmi.sock`, then start the Mapper
again. EdgeCore replaces its Unix socket during restart, so a new Mapper registration
is required.

```bash
systemctl restart edgecore
systemctl is-active edgecore
ls -l /etc/kubeedge/dmi.sock
journalctl -u edgecore --since "2 minutes ago" --no-pager |
  grep -Ei 'dmi worker|init device|device model|DMI Server|failed'
```

## 5. `slave_address must be an integer, got 1.0`

KubeEdge `CustomizedValue` uses protobuf `Any`; some CRD conversion paths encode a
YAML number as `FloatValue`, producing `"slave_address": 1.0`. Put integer settings
in `configData` explicitly in strings:

```yaml
configData:
  port: /dev/ttyUSB0
  slave_address: "1"
  baud_rate: "9600"
  data_bits: "8"
  parity: none
  stop_bits: "1"
  timeout_seconds: 1.0
  retries: "2"
  probe_on_connect: true
  verify_writes: true
  handle_local_echo: false
```

`timeout_seconds` is genuinely floating point. Reapply the resource; a transient
old `DEVICE_RECONNECT_FAILED` may precede the successful new configuration.

## 6. Empty `desired: {}` is treated as a write

CRD-to-DMI conversion may produce an empty desired protobuf message even without
desired in YAML. Treat desired as present only when it has a value or metadata:

```python
desired_present = prop.HasField("desired") and bool(
    prop.desired.value or prop.desired.metadata
)
```

This ignores empty desired while retaining valid values such as `"0"`. After this
framework change, restart Mapper; a Device without desired must not print empty
`CLOUD_DESIRED_VALUE` or fail numeric conversion.

## 7. Serial and PTY checks

Enter this layer only if the Device arrived and `DEVICE_START_PENDING.error` points
to a serial or Modbus request. With simulator `--create-pty`, configure the Device
with the Mapper-side PTY printed by the simulator, not the other endpoint. PTY
numbers change after restart.

- Cannot open port: path missing, wrong endpoint, or insufficient permissions.
- No register reply: simulator is stopped, PTYs reversed, address/baud mismatch.
- Probe fails after connect: verify simulator ownership of its PTY and slave address.

## 8. Final verification

First expect `MAPPER_REGISTER_RESPONSE` with Device/Model lists, then
`REGISTER_DEVICE`, `DEVICE_RECONNECTED`, or `UPDATE_DEVICE`. Finally run:

```bash
curl http://127.0.0.1:7777/api/v1/ping
curl http://127.0.0.1:7777/api/v1/devicemethod/default/industrial-thermostat
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/temperature_pv
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/humidity_rh
curl http://127.0.0.1:7777/api/v1/device/default/industrial-thermostat/alarm_status
```

The property reads should return 200 only after a successful Device-session event.
