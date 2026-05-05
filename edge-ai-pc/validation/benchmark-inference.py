#!/usr/bin/env python3
"""
benchmark-inference.py — Inference latency & throughput benchmark for KubeEdge AI PC
Measures P50/P95/P99 latency and throughput for:
  - OpenVINO Model Server (OVMS) gRPC endpoint (vision models)
  - llama.cpp HTTP server (LLM models)

Usage:
  pip install grpcio tritonclient[grpc] requests numpy
  python3 benchmark-inference.py --target ovms  --endpoint 127.0.0.1:9000 --model yolov8n --runs 200
  python3 benchmark-inference.py --target llama --endpoint http://127.0.0.1:8080 --runs 50
"""

import argparse
import json
import time
import random
import statistics
import sys
import numpy as np

try:
    import requests
except ImportError:
    print("pip install requests"); sys.exit(1)

# ─── Utility ──────────────────────────────────────────────────────────────────

def percentile(data, pct):
    data_sorted = sorted(data)
    idx = int(len(data_sorted) * pct / 100)
    return data_sorted[min(idx, len(data_sorted) - 1)]

def print_stats(label, latencies_ms):
    p50  = percentile(latencies_ms, 50)
    p95  = percentile(latencies_ms, 95)
    p99  = percentile(latencies_ms, 99)
    mean = statistics.mean(latencies_ms)
    mn   = min(latencies_ms)
    mx   = max(latencies_ms)
    throughput = 1000.0 / mean  # requests per second

    print(f"\n{'='*55}")
    print(f"  {label}")
    print(f"{'='*55}")
    print(f"  Runs     : {len(latencies_ms)}")
    print(f"  Min      : {mn:.1f} ms")
    print(f"  Mean     : {mean:.1f} ms")
    print(f"  P50      : {p50:.1f} ms")
    print(f"  P95      : {p95:.1f} ms")
    print(f"  P99      : {p99:.1f} ms")
    print(f"  Max      : {mx:.1f} ms")
    print(f"  RPS      : {throughput:.2f} req/s")
    print(f"{'='*55}\n")

# ─── OVMS gRPC benchmark ──────────────────────────────────────────────────────

def benchmark_ovms_rest(endpoint: str, model: str, runs: int, warmup: int):
    """Benchmark OVMS via REST /v2/models/<model>/infer"""
    import base64
    url = f"http://{endpoint}/v2/models/{model}/infer"
    # Generate a random 640x640x3 image (simulated camera frame)
    dummy_input = np.random.rand(1, 3, 640, 640).astype(np.float32).flatten().tolist()
    payload = {
        "inputs": [{"name": "images", "shape": [1, 3, 640, 640], "datatype": "FP32", "data": dummy_input}]
    }
    headers = {"Content-Type": "application/json"}

    print(f"Warming up ({warmup} requests) against {url}...")
    for _ in range(warmup):
        try:
            requests.post(url, json=payload, headers=headers, timeout=10)
        except Exception:
            pass

    print(f"Benchmarking {runs} inference requests...")
    latencies = []
    errors = 0
    for i in range(runs):
        t0 = time.perf_counter()
        try:
            resp = requests.post(url, json=payload, headers=headers, timeout=30)
            elapsed = (time.perf_counter() - t0) * 1000
            if resp.status_code == 200:
                latencies.append(elapsed)
            else:
                errors += 1
                print(f"  Request {i+1}: HTTP {resp.status_code}")
        except Exception as e:
            errors += 1
            print(f"  Request {i+1}: ERROR {e}")

    print(f"  Errors: {errors}/{runs}")
    if latencies:
        print_stats(f"OVMS REST | model={model} | {endpoint}", latencies)
    return latencies

# ─── llama.cpp HTTP server benchmark ──────────────────────────────────────────

def benchmark_llama(endpoint: str, runs: int, warmup: int):
    """Benchmark llama.cpp /completion endpoint"""
    url = f"{endpoint}/completion"
    prompts = [
        "Summarize edge computing in one sentence.",
        "What is an NPU?",
        "Explain KubeEdge in 20 words.",
        "What makes AI PCs suitable for edge inference?",
        "Describe the Phi-3 model architecture briefly.",
    ]
    payload_base = {
        "n_predict": 64,
        "temperature": 0.0,
        "top_k": 1,
        "stop": ["\n"],
    }
    headers = {"Content-Type": "application/json"}

    print(f"Warming up ({warmup} requests) against {url}...")
    for _ in range(warmup):
        try:
            p = {**payload_base, "prompt": random.choice(prompts)}
            requests.post(url, json=p, headers=headers, timeout=60)
        except Exception:
            pass

    print(f"Benchmarking {runs} LLM completion requests (n_predict=64)...")
    latencies = []
    token_counts = []
    errors = 0
    for i in range(runs):
        prompt = random.choice(prompts)
        payload = {**payload_base, "prompt": prompt}
        t0 = time.perf_counter()
        try:
            resp = requests.post(url, json=payload, headers=headers, timeout=120)
            elapsed = (time.perf_counter() - t0) * 1000
            if resp.status_code == 200:
                data = resp.json()
                latencies.append(elapsed)
                tokens = data.get("tokens_evaluated", 0) + data.get("tokens_predicted", 0)
                token_counts.append(tokens)
            else:
                errors += 1
        except Exception as e:
            errors += 1
            print(f"  Request {i+1}: ERROR {e}")

    print(f"  Errors: {errors}/{runs}")
    if latencies and token_counts:
        print_stats(f"llama.cpp | {endpoint}", latencies)
        avg_tokens = statistics.mean(token_counts)
        avg_tps = avg_tokens / (statistics.mean(latencies) / 1000)
        print(f"  Avg tokens/request : {avg_tokens:.0f}")
        print(f"  Avg tokens/second  : {avg_tps:.1f} tok/s")
    return latencies

# ─── Main ─────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(description="KubeEdge AI PC Inference Benchmark")
    parser.add_argument("--target",   choices=["ovms", "llama"], required=True)
    parser.add_argument("--endpoint", default="127.0.0.1:9000",
                        help="OVMS: host:port | llama: http://host:port")
    parser.add_argument("--model",    default="yolov8n", help="OVMS model name")
    parser.add_argument("--runs",     type=int, default=100)
    parser.add_argument("--warmup",   type=int, default=5)
    args = parser.parse_args()

    if args.target == "ovms":
        benchmark_ovms_rest(args.endpoint, args.model, args.runs, args.warmup)
    elif args.target == "llama":
        endpoint = args.endpoint
        if not endpoint.startswith("http"):
            endpoint = f"http://{endpoint}"
        benchmark_llama(endpoint, args.runs, args.warmup)

if __name__ == "__main__":
    main()
