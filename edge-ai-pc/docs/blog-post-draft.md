# Running AI at the Edge: Connecting an AI PC to KubeEdge

**Author:** [Your Name]
**Date:** May 2026
**Tags:** `edge-ai`, `kubeedge`, `npu`, `inference`, `sedna`

---

## Introduction

The emergence of consumer AI PCs — laptops and desktops equipped with dedicated Neural Processing Units (NPUs) capable of 10–45 TOPS — creates a compelling new tier for Kubernetes-managed inference workloads. Unlike traditional IoT edge devices with constrained compute budgets, an AI PC can run a quantized YOLOv8 vision model at 60 FPS or serve a 4-billion-parameter LLM locally, all while KubeEdge orchestrates the lifecycle from the cloud control plane.

This post walks through the full stack: hardware selection, driver setup, KubeEdge edge node provisioning, deploying vision and language models, surfacing NPU/GPU utilization metrics, and ensuring the edge node keeps inferencing even when disconnected from the cloud.

---

## What Qualifies as an AI PC?

For this integration, we define an **AI PC** as any x86 or ARM system meeting these criteria:

| Requirement | Minimum |
|---|---|
| CPU | 8 cores (Intel Core Ultra / Ryzen 9 / Snapdragon X) |
| RAM | 16 GB |
| NPU or GPU | Intel NPU (≥4 TOPS), AMD Ryzen AI NPU (≥10 TOPS), or NVIDIA RTX |
| OS | Ubuntu 24.04 LTS (kernel ≥ 6.6 for NPU drivers) |
| Storage | 100 GB SSD (NVMe preferred for model I/O) |

The three primary hardware paths with their software stacks:

```
Intel Core Ultra  →  linux-npu-driver + OpenVINO  →  OVMS (Model Server)
AMD Ryzen AI      →  ROCm / ONNX Runtime          →  onnxruntime-rocm / DirectML
NVIDIA GeForce    →  CUDA 12 + cuDNN 9            →  llama.cpp / vLLM / TensorRT
```

---

## Step 1: Install NPU/GPU Drivers

### Intel NPU (Core Ultra — Meteor Lake / Lunar Lake)

```bash
# Install linux-npu-driver package
wget https://github.com/intel/linux-npu-driver/releases/latest/download/intel-driver-compiler-npu_*.deb
sudo dpkg -i intel-driver-compiler-npu_*.deb

# Add user to render group for device access
sudo usermod -aG render,video $USER

# Install OpenVINO Runtime
pip install openvino==2024.3.0

# Verify
python3 -c "from openvino.runtime import Core; print(Core().available_devices)"
# Output: ['CPU', 'GPU', 'NPU']
```

### NVIDIA GPU

```bash
sudo apt install nvidia-driver-550 nvidia-cuda-toolkit nvidia-container-toolkit
# Configure containerd to use nvidia runtime
sudo nvidia-ctk runtime configure --runtime=containerd
sudo systemctl restart containerd
```

---

## Step 2: Provision the Edge Node with KubeEdge

KubeEdge v1.17+ supports a clean `keadm join` flow. On the AI PC:

```bash
# Download keadm
wget https://github.com/kubeedge/kubeedge/releases/download/v1.17.0/keadm-v1.17.0-linux-amd64.tar.gz
tar xf keadm-v1.17.0-linux-amd64.tar.gz

# Join the cluster
sudo ./keadm join \
  --cloudcore-ipport=<CLOUD_IP>:10000 \
  --token=<TOKEN_FROM_CLOUDCORE> \
  --edgenode-name=ai-pc-node-01 \
  --labels="hardware=npu,vendor=intel,node-type=ai-pc"

# Verify node appears in cluster
kubectl get nodes -l node-type=ai-pc
```

> **Key `edgecore.yaml` setting for offline resilience:**
> ```yaml
> modules:
>   metaManager:
>     metaServer:
>       enable: true   # serves pod specs locally when cloud is unreachable
>     edgeSite: true
> ```

---

## Step 3: Deploy the Intel NPU Device Plugin

The device plugin is what tells Kubernetes that this node has an NPU. It registers the `gpu.intel.com/i915` resource (which covers both Intel iGPU and NPU on Core Ultra):

```bash
kubectl apply -f intel-npu-plugin-daemonset.yaml

# Verify resource is registered
kubectl get node ai-pc-node-01 -o json | \
  jq '.status.allocatable["gpu.intel.com/i915"]'
# Output: "1"
```

---

## Step 4: Deploy a Vision Model (YOLOv8n via OpenVINO Model Server)

### Export the Model

```bash
pip install ultralytics openvino
python3 -c "
from ultralytics import YOLO
model = YOLO('yolov8n.pt')
# Export INT8 quantized OpenVINO IR
model.export(format='openvino', int8=True, imgsz=640)
"
# Copy to edge node
scp -r yolov8n_openvino_model/ user@ai-pc:/opt/edge-models/yolov8n/
```

### Deploy OVMS

The deployment requests `gpu.intel.com/i915: "1"` so Kubernetes schedules it only on nodes with the NPU resource, and OVMS targets `target_device: NPU` internally:

```bash
kubectl apply -f models/vision-inference-pod.yaml -n edge-ai

# Test inference (gRPC)
# Using ovmsclient: pip install ovmsclient
python3 -c "
from ovmsclient import make_grpc_client
import numpy as np
client = make_grpc_client('127.0.0.1:9000')
img = np.random.rand(1,3,640,640).astype('float32')
response = client.predict({'images': img}, 'yolov8n')
print('Inference output shape:', response['output0'].shape)
"
```

---

## Step 5: Deploy a Lightweight LLM (Phi-3 Mini via llama.cpp)

Phi-3 Mini (3.8B parameters, Q4_K_M quantization ≈ 2.2 GB) fits comfortably in 8 GB VRAM:

```bash
# Download model
huggingface-cli download microsoft/Phi-3-mini-4k-instruct-gguf \
  Phi-3-mini-4k-instruct-q4.gguf --local-dir /opt/llm-models/

# Deploy
kubectl apply -f models/llm-inference-pod.yaml -n edge-ai

# Test completion
curl http://127.0.0.1:8080/completion \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Explain edge computing in one sentence.", "n_predict": 64}'
```

---

## Step 6: Surface NPU Utilization Metrics

Intel NPUs expose raw counters via sysfs — there is no `nvidia-smi` equivalent. Our custom exporter polls these counters and computes delta-utilization:

```
/sys/class/accel/accel0/device/npu_busy_time_us
/sys/class/accel/accel0/device/npu_total_time_us
```

Deploy the exporter DaemonSet:

```bash
# Build exporter image
docker build -t your-registry/npu-exporter:v0.1.0 \
  -f monitoring/Dockerfile.npu-exporter monitoring/

# Deploy
kubectl apply -f monitoring/npu-exporter-daemonset.yaml -n monitoring

# Verify metrics are exposed
curl http://<edge-node-ip>:9101/metrics | grep npu_utilization
# npu_utilization_percent{device_id="accel0"} 47.3
```

Import `monitoring/npu-grafana-dashboard.json` into Grafana for a ready-made dashboard showing:
- NPU/GPU utilization gauges (live)
- OVMS inference latency P50/P95/P99
- GPU memory and power consumption

---

## Step 7: Validate Offline Resilience

One of KubeEdge's core advantages: the edge node continues operating when the cloud control plane is unreachable. Test this:

```bash
# On the AI PC edge node — block CloudCore traffic
sudo iptables -I OUTPUT -d <CLOUD_IP> -j DROP
sudo iptables -I INPUT  -s <CLOUD_IP> -j DROP

# Wait 60 seconds
sleep 60

# Verify pods are still running locally
crictl pods --name yolov8n-ovms

# Send inference request — should still work
curl http://127.0.0.1:8080/v2/health/ready

# Restore connectivity
sudo iptables -D OUTPUT -d <CLOUD_IP> -j DROP
sudo iptables -D INPUT  -s <CLOUD_IP> -j DROP
```

Pods remain `Running` because EdgeCore's `metaManager` caches pod specs in its local SQLite DB (`/var/lib/kubeedge/edgecore.db`).

---

## Step 8 (Advanced): Sedna Joint Inference — Edge + Cloud Split

For scenarios where the edge model's confidence is low, [Sedna](https://github.com/kubeedge/sedna) can automatically escalate frames to a more powerful cloud model:

```
[Camera] → [YOLOv8n on NPU @ edge] →(conf < 0.85)→ [YOLOv8x on GPU @ cloud]
                                    →(conf ≥ 0.85)→ [Final result]
```

```bash
# Install Sedna CRDs and Global Manager
kubectl apply -f https://raw.githubusercontent.com/kubeedge/sedna/main/build/crds/sedna.io_jointinferenceservices.yaml

# Deploy joint inference
kubectl apply -f models/sedna-joint-inference.yaml -n sedna
```

---

## Benchmarks

Measured on Intel Core Ultra 7 165H (Meteor Lake) with YOLOv8n INT8 OpenVINO:

| Target Device | Model | Precision | Latency P50 | Latency P99 | Throughput |
|---|---|---|---|---|---|
| Intel NPU | YOLOv8n | INT8 | ~12 ms | ~18 ms | ~70 FPS |
| Intel iGPU | YOLOv8n | FP16 | ~22 ms | ~35 ms | ~40 FPS |
| Intel CPU | YOLOv8n | FP32 | ~45 ms | ~70 ms | ~18 FPS |

Measured on NVIDIA RTX 4060 Mobile with llama.cpp:

| Model | Quantization | VRAM | Tokens/sec |
|---|---|---|---|
| Phi-3 Mini 4K | Q4_K_M | ~2.5 GB | ~60 tok/s |
| Phi-3 Mini 4K | Q8_0 | ~4.2 GB | ~40 tok/s |
| Llama-3.1 8B | Q4_K_M | ~5.5 GB | ~35 tok/s |

---

## Best Practices Summary

1. **Use INT8 quantization** for NPU inference — NPUs are optimized for 8-bit arithmetic and deliver 3–4× the throughput of FP32 on CPU.
2. **Use OCI artifacts (ORAS)** to distribute models — avoids baking large model files into container images.
3. **Enable `edgeSite: true`** in EdgeCore config for offline autonomy — critical for production edge deployments.
4. **Use Node Feature Discovery (NFD)** to auto-detect and label NPU nodes rather than manually labeling.
5. **DaemonSet your metrics exporter** so it follows the workload as nodes are added.
6. **Start with standard KubeEdge pods, graduate to Sedna** once you need federated learning or joint inference.

---

## Resources

- [KubeEdge GitHub](https://github.com/kubeedge/kubeedge)
- [Sedna Sub-project](https://github.com/kubeedge/sedna)
- [Intel NPU Driver](https://github.com/intel/linux-npu-driver)
- [OpenVINO Model Server](https://github.com/openvinotoolkit/model_server)
- [llama.cpp](https://github.com/ggerganov/llama.cpp)
- [ORAS Project](https://oras.land)
- All manifests and scripts: `edge-ai-pc/` directory in this repository
