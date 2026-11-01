# Multi-Architecture Lightweight LLM Validation Report

This report records the midterm validation results for lightweight LLM
inference on KubeEdge-managed edge nodes. It follows the validation plan in
[PR #6907](https://github.com/kubeedge/kubeedge/pull/6907) and contributes to
[issue #6756](https://github.com/kubeedge/kubeedge/issues/6756).

| Item | Value |
| --- | --- |
| Validation date | 2026-07-13 |
| KubeEdge version | v1.23.0 |
| Model runtime | Ollama |
| Node environments | x86 Linux, ARM64 Jetson, Windows WSL x86_64 |
| Models | SmolLM2 135M, Qwen2.5 0.5B, TinyLlama 1.1B |

## Summary

The validation used one Kubernetes control plane with KubeEdge CloudCore to
manage three heterogeneous edge nodes. All three nodes joined the cluster,
became `Ready`, received workloads targeted by the control plane, ran Ollama,
and completed inference with all three models.

The Windows WSL node was additionally limited to 4 GB RAM and 1 GB swap. After
the resource limit was applied and the node restarted, EdgeCore, containerd,
and the Ollama workload recovered, and the three models completed inference
sequentially.

These results validate multi-architecture compatibility and the basic
deployment and operation workflow. They are not a cross-node performance
benchmark. Model quantization, prompt length, and other runtime conditions were
not fully normalized across the three environments.

## Validation Scope

The following items were validated:

- edge node onboarding and `Ready` status;
- targeted `Job` and `Deployment` delivery from the control plane;
- Ollama startup and persistent model storage;
- loading and inference for three lightweight LLMs;
- remote `kubectl exec` and `kubectl logs` on the ARM64 and WSL nodes;
- sequential model inference under a 4 GB RAM and 1 GB swap WSL limit.

The following items were not validated in this stage:

- normalized cross-node model performance;
- concurrent requests or multiple models remaining loaded at the same time;
- long-running stability or production service-level objectives;
- systematic ClusterIP, DNS, CNI, NodePort, or EdgeMesh testing;
- long-duration disconnect, reconnect, and high-availability behavior;
- production security, isolation, or capacity planning.

## Test Architecture

The cloud node ran the Kubernetes control plane and KubeEdge CloudCore. The
control plane created workloads with node selectors targeting the required edge
node. EdgeCore received the workload metadata, and containerd started the
containers on each edge node.

The ARM64 Jetson and WSL nodes were in different physical networks from the
cloud node. A ZeroTier virtual network provided the required CloudCore and API
connectivity. Network identifiers, tokens, and complete internal addresses are
intentionally omitted from this report.

## Test Environments

| Item | x86 Linux | ARM64 Jetson | Windows WSL |
| --- | --- | --- | --- |
| Node architecture | amd64 | arm64 | amd64 (WSL2) |
| Operating system | Ubuntu 22.04.4 | Ubuntu 18.04.5 | Ubuntu 22.04.5 |
| KubeEdge version | v1.23.0 | v1.23.0 | v1.23.0 |
| Container runtime | containerd 2.2.1 | containerd 1.7.7 | containerd 2.2.1 |
| Accelerator | CPU only | Jetson GPU present, not used | CPU only |
| Memory constraint | 7.6 GiB observed | 3.9 GiB observed | 4 GB RAM + 1 GB swap |
| Workload type | `edge-llm-test` Job | Ollama Deployment | Ollama Deployment |
| Final node state | `Ready` | `Ready` | `Ready` |

The Ollama Deployments used `nodeSelector` for placement, `hostNetwork` for
local port `11434`, and `hostPath` volumes for `/data/ollama` and
`/data/models`. `OLLAMA_MAX_LOADED_MODELS=1` was used to avoid keeping multiple
models loaded simultaneously on resource-constrained nodes.

## Validation Method

The experiment used the following workflow on each node:

1. Join the edge node to CloudCore with an architecture-matched `keadm`
   binary.
2. Confirm that the node is `Ready` and has the `agent,edge` role.
3. Deliver a workload from the control plane to the selected node.
4. Start Ollama and verify its model list through `/api/tags` or `ollama list`.
5. Run SmolLM2 135M, Qwen2.5 0.5B, and TinyLlama 1.1B sequentially.
6. Confirm inference completion and observe CPU, memory, swap, and accelerator
   usage.
7. For ARM64 and WSL, verify remote operations through `kubectl exec` and
   `kubectl logs`.

Representative evidence commands were:

```bash
kubectl get nodes -o wide
kubectl -n ai-edge get pods -o wide
kubectl -n ai-edge exec deploy/ollama-edge -- ollama list
kubectl -n ai-edge logs deploy/ollama-edge
curl -s http://127.0.0.1:11434/api/tags
```

## Validation Results

### Overall Compatibility Matrix

| Validation item | x86 Linux | ARM64 Jetson | Windows WSL |
| --- | --- | --- | --- |
| Node joined and `Ready` | Passed | Passed | Passed |
| Workload delivery | Job succeeded | Deployment running | Deployment running |
| SmolLM2 135M inference | Passed | Passed | Passed |
| Qwen2.5 0.5B inference | Passed | Passed | Passed |
| TinyLlama 1.1B inference | Passed | Passed | Passed |
| Remote operations | Job logs read from control plane | `exec` and `logs` passed | `exec` and `logs` passed |
| Resource-constrained run | Not tested | Not tested | Passed with 4 GB RAM + 1 GB swap |

### x86 Linux

The control plane delivered the `edge-llm-test` Job to the x86 edge node. The
Job queried the local Ollama API and ran all three models. It completed with
`Succeeded`, and its logs recorded successful responses with `done: true`.

| Model | Result | Observed resource data |
| --- | --- | --- |
| SmolLM2 135M | Passed | Runner CPU peaked at about 96.4%; RSS was about 766 MB |
| Qwen2.5 0.5B | Passed | Main runner used about 135% CPU and 766 MB RSS; a second observed runner used about 9.7% CPU and 577 MB RSS |
| TinyLlama 1.1B | Passed | Main runner used about 114% CPU and 840 MB RSS; a second observed runner used about 6.8% CPU and 749 MB RSS |

The node had 7.6 GiB total memory and did not use swap during the recorded
runs. CPU values above 100% indicate multi-core CPU use.

### ARM64 Jetson

The `ollama-edge` Deployment remained `1/1 Running` on the Jetson node. All
three models were available through Ollama and completed inference. The GPU
utilization remained at 0%, so these results represent CPU inference rather
than Jetson GPU acceleration.

| Model | Result | Observed resource data |
| --- | --- | --- |
| SmolLM2 135M | Passed | CPU cores reached about 79%-100%; system memory was about 931 MB; swap did not increase noticeably |
| Qwen2.5 0.5B | Passed | About 29.43 seconds end-to-end and 9.94 tokens/s in one recorded run; CPU was about 94%-100%; memory was about 1,309 MB; swap was about 125 MB |
| TinyLlama 1.1B | Passed | CPU was about 91%-100%; memory was about 1,447 MB of 3,964 MB; swap was about 137 MB of 1,982 MB |

The Qwen2.5 timing is a single observed run, not a p50 or p95 measurement.

### Windows WSL Under a 4 GB Memory Limit

The WSL node first passed basic workload delivery with a pause Pod. The control
plane then delivered the `ollama-wsl-edge` Deployment. After applying a 4 GB
RAM and 1 GB swap limit in `.wslconfig`, the WSL environment was restarted.
EdgeCore, containerd, and the Ollama workload recovered, and all three models
completed inference sequentially.

| Model | Result | Observed resource data |
| --- | --- | --- |
| SmolLM2 135M | Passed | System memory used about 738 MiB; runner used about 115% CPU and 196 MB RSS; swap remained unused |
| Qwen2.5 0.5B | Passed | System memory used about 1.0 GiB; runner used about 100% CPU and 505 MB RSS; swap remained unused |
| TinyLlama 1.1B | Passed | System memory used about 1.2 GiB; runner used about 73% CPU and 693 MB RSS; swap remained unused |

This result establishes sequential execution under the configured memory
limit. It does not establish concurrency capacity or a long-running memory
stability limit.

### Output Quality Observation

Model answer quality was not a primary acceptance criterion. In the recorded
Chinese edge-computing prompt, Qwen2.5 0.5B produced the most consistently
relevant Chinese answer. SmolLM2 135M sometimes produced unrelated content,
and TinyLlama 1.1B was less consistent across nodes. These observations are
informational only because prompts and model formats were not fully normalized.

## Problems Found and Resolutions

| Problem | Impact | Resolution | Result |
| --- | --- | --- | --- |
| Edge nodes and CloudCore were on different physical networks | ARM64 and WSL nodes could not reliably reach CloudCore | Connected the nodes through a ZeroTier virtual network | Control-channel and API connectivity recovered |
| Image registry access timed out | Ollama images or installation artifacts could not be pulled | Used an accessible mirror and pre-pulled images on edge nodes | Pods started successfully |
| `kubectl exec` and `kubectl logs` returned `InternalError` on WSL | The control plane could not remotely operate the edge Pod | Enabled `edgeStream` in `edgecore.yaml` and restarted EdgeCore | Both commands succeeded |
| Default WSL memory did not represent a constrained node | The low-memory objective could not be evaluated | Set WSL2 to 4 GB RAM and 1 GB swap, then restarted it | All three models ran sequentially |
| Local TinyLlama GGUF import stalled | The model could not be created in Ollama | Used the packaged `tinyllama:1.1b` model layers | The model was listed and completed inference |
| Rolling update left a `hostNetwork` Pod pending | Old and new Pods competed for port `11434` on the same node | Configure the Deployment with `strategy.type: Recreate` so the old Pod stops before its replacement starts | In this run, manually terminating the old Pod allowed the replacement to reach `1/1 Running`; `Recreate` is recommended for future runs |

## Conclusions

KubeEdge v1.23.0 successfully managed lightweight LLM workloads in three
heterogeneous edge environments:

- **Node compatibility:** x86 Linux, ARM64 Jetson, and Windows WSL x86_64 all
  joined the same KubeEdge system and remained `Ready`.
- **Workload delivery:** the control plane delivered a Job or Deployment to
  each selected edge node.
- **Inference compatibility:** SmolLM2 135M, Qwen2.5 0.5B, and TinyLlama 1.1B
  completed inference on every node.
- **Remote operations:** the control plane accessed ARM64 and WSL Pods through
  `kubectl exec` and `kubectl logs` after EdgeStream was enabled.
- **Low-memory feasibility:** all three models ran sequentially on WSL with
  approximately 3.8 GiB visible memory and 1 GiB swap.

The result demonstrates a working multi-architecture deployment and operation
path. It should not be interpreted as a model ranking, an accelerator
benchmark, or a production capacity statement.

## Limitations and Next Steps

The current run used an unpinned Ollama image and may have used different model
quantization formats on different nodes. Prompt lengths and network conditions
also differed. These factors limit strict reproducibility and cross-node
comparison.

### Reproducibility

This report provides the environment matrix, validation workflow,
representative commands, and observed results. It does not yet include the
exact manifests and scripts used for every run because the Ollama image digest,
model artifacts, quantization formats, and prompts were not pinned consistently
across all three nodes. The current report can therefore be used to reproduce
the deployment workflow, but not an identical benchmark run.

The next stage will add versioned manifests and validation scripts after the
image digest, model artifact versions and checksums, quantization formats, and
prompt set are fixed. Raw outputs and metric samples will also be retained where
practical so that the reported results can be traced to their source data.

The next validation stage should:

1. Pin the Ollama image digest and model artifacts.
2. Use one fixed prompt set and a fixed output-token limit.
3. Separate cold start, model load, and warm inference measurements.
4. Repeat each test and report p50 and p95 latency.
5. Collect synchronized CPU, memory, disk, swap, temperature, and accelerator
   metrics.
6. Validate concurrent requests and long-running stability.
7. Test disconnect and reconnect recovery, offline image distribution, and
   model cache persistence.
8. Validate ClusterIP, DNS, NodePort, edge-local access, and EdgeMesh where
   applicable.
9. Enable and measure Jetson GPU acceleration separately from the current CPU
   baseline.
