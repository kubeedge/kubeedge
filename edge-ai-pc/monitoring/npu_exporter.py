#!/usr/bin/env python3
"""
npu_exporter.py — Custom Prometheus Exporter for Intel NPU
=============================================================
Reads utilization and timing counters from the Intel NPU sysfs interface
(linux-npu-driver >= 1.5) and exposes them as Prometheus metrics.

Metrics exposed (port 9101):
  npu_utilization_percent        - NPU compute utilization (0-100%)
  npu_busy_time_us_total         - Cumulative NPU busy time in microseconds
  npu_total_time_us_total        - Cumulative NPU total time in microseconds
  npu_device_info                - Static info (driver version, device ID)
  npu_scrape_errors_total        - Counter of sysfs read errors

Usage:
  pip install prometheus-client
  python3 npu_exporter.py [--port 9101] [--accel-id 0] [--interval 5]

Container usage:
  docker run --privileged -v /sys:/sys -p 9101:9101 npu-exporter:v0.1.0
"""

import argparse
import pathlib
import time
import logging
import sys
import os
from prometheus_client import (
    start_http_server,
    Gauge,
    Counter,
    Info,
    REGISTRY,
    CollectorRegistry,
)
from prometheus_client.core import GaugeMetricFamily, CounterMetricFamily

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    stream=sys.stdout,
)
log = logging.getLogger("npu-exporter")

# ─── Prometheus metric definitions ────────────────────────────────────────────

NPU_UTIL = Gauge(
    "npu_utilization_percent",
    "Intel NPU compute utilization percentage (delta over scrape interval)",
    ["device_id"],
)
NPU_BUSY = Counter(
    "npu_busy_time_us",
    "Cumulative microseconds the NPU was busy executing workloads",
    ["device_id"],
)
NPU_TOTAL = Counter(
    "npu_total_time_us",
    "Cumulative total time microseconds since NPU driver start",
    ["device_id"],
)
NPU_SCRAPE_ERRORS = Counter(
    "npu_scrape_errors_total",
    "Number of sysfs read errors encountered by the exporter",
    ["device_id", "counter"],
)
NPU_INFO = Info(
    "npu_device",
    "Static Intel NPU device information",
)


def get_sysfs_value(path: pathlib.Path) -> int:
    """Read an integer from a sysfs file. Returns -1 on failure."""
    try:
        return int(path.read_text().strip())
    except (FileNotFoundError, ValueError, PermissionError) as e:
        log.debug("sysfs read failed [%s]: %s", path, e)
        return -1


def get_sysfs_str(path: pathlib.Path, default: str = "unknown") -> str:
    """Read a string from a sysfs file."""
    try:
        return path.read_text().strip()
    except Exception:
        return default


def collect_device_info(accel_path: pathlib.Path, device_id: str) -> None:
    """Populate static NPU device info metrics."""
    driver_ver = get_sysfs_str(accel_path / "driver_version")
    device_name = get_sysfs_str(accel_path / "device_name", default="Intel NPU")
    vendor_id   = get_sysfs_str(accel_path.parent / "vendor")
    pci_id      = get_sysfs_str(accel_path.parent / "device")

    NPU_INFO.info({
        "device_id":     device_id,
        "device_name":   device_name,
        "driver_version": driver_ver,
        "vendor_id":     vendor_id,
        "pci_device_id": pci_id,
    })
    log.info("NPU device: %s | driver: %s | PCI: %s", device_name, driver_ver, pci_id)


class NPUMetricsCollector:
    """
    Stateful collector that computes delta-utilization between scrapes.
    Stores previous busy/total counters to compute utilization %.
    """

    def __init__(self, accel_id: int, interval: float):
        self.accel_id = accel_id
        self.device_id = f"accel{accel_id}"
        self.interval = interval
        self.accel_path = pathlib.Path(f"/sys/class/accel/accel{accel_id}/device")

        # Snapshot state for delta calculation
        self._prev_busy: int = -1
        self._prev_total: int = -1

        # One-time device info
        collect_device_info(self.accel_path, self.device_id)

    def _read_counters(self) -> tuple[int, int]:
        """Returns (busy_us, total_us) or (-1, -1) on failure."""
        busy  = get_sysfs_value(self.accel_path / "npu_busy_time_us")
        total = get_sysfs_value(self.accel_path / "npu_total_time_us")
        return busy, total

    def collect(self) -> None:
        """Called every `interval` seconds to update Prometheus metrics."""
        busy, total = self._read_counters()

        if busy == -1 or total == -1:
            log.warning("Failed to read NPU counters for %s", self.device_id)
            NPU_SCRAPE_ERRORS.labels(
                device_id=self.device_id, counter="busy_or_total"
            ).inc()
            NPU_UTIL.labels(device_id=self.device_id).set(-1)
            return

        # Update cumulative counters (prometheus_client Counter wraps correctly)
        # Note: Counter.inc() only — we track absolute via Gauge for util
        if self._prev_busy >= 0 and self._prev_total >= 0:
            delta_busy  = max(busy  - self._prev_busy,  0)
            delta_total = max(total - self._prev_total, 1)  # avoid div-by-zero
            util = (delta_busy / delta_total) * 100.0
            util = min(max(util, 0.0), 100.0)  # clamp
            NPU_UTIL.labels(device_id=self.device_id).set(util)
            log.debug("NPU[%s] util=%.1f%% (Δbusy=%d Δtotal=%d)",
                      self.device_id, util, delta_busy, delta_total)
        else:
            # First scrape — no delta yet
            NPU_UTIL.labels(device_id=self.device_id).set(0.0)
            log.info("NPU[%s] first scrape — baseline established.", self.device_id)

        self._prev_busy  = busy
        self._prev_total = total


def discover_accel_devices() -> list[int]:
    """Auto-discover Intel accel devices from /sys/class/accel/."""
    accel_base = pathlib.Path("/sys/class/accel")
    if not accel_base.exists():
        log.warning("/sys/class/accel not found. No NPU devices detected.")
        return []
    devices = sorted(
        int(p.name.replace("accel", ""))
        for p in accel_base.iterdir()
        if p.name.startswith("accel") and p.name[5:].isdigit()
    )
    log.info("Discovered %d NPU accel device(s): %s", len(devices), devices)
    return devices


def main():
    parser = argparse.ArgumentParser(description="Intel NPU Prometheus Exporter")
    parser.add_argument("--port",     type=int,   default=9101,  help="Metrics HTTP port")
    parser.add_argument("--accel-id", type=int,   default=None,  help="Specific accel device ID (default: auto-discover)")
    parser.add_argument("--interval", type=float, default=5.0,   help="Scrape interval in seconds")
    parser.add_argument("--debug",    action="store_true",        help="Enable debug logging")
    args = parser.parse_args()

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)

    # Device discovery
    if args.accel_id is not None:
        accel_ids = [args.accel_id]
    else:
        accel_ids = discover_accel_devices()
        if not accel_ids:
            log.error("No NPU accel devices found. Exiting.")
            sys.exit(1)

    # Initialize collectors for each device
    collectors = [NPUMetricsCollector(aid, args.interval) for aid in accel_ids]

    # Start Prometheus HTTP server
    start_http_server(args.port)
    log.info("NPU exporter started on :%d (scrape interval: %.1fs)", args.port, args.interval)
    log.info("Metrics endpoint: http://0.0.0.0:%d/metrics", args.port)

    # Main collection loop
    while True:
        for collector in collectors:
            collector.collect()
        time.sleep(args.interval)


if __name__ == "__main__":
    main()
