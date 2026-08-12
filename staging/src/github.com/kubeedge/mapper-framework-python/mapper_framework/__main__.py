from __future__ import annotations

import argparse
import logging

from .config import Config
from .events import emit
from .runtime import MapperRuntime, install_signal_handlers, load_driver_type


def main() -> None:
    parser = argparse.ArgumentParser(description="KubeEdge Python Mapper Framework")
    parser.add_argument("--config", default="config.yaml", help="mapper YAML configuration")
    parser.add_argument(
        "--driver",
        action="append",
        default=[],
        help="custom driver in MODULE:CLASS format; may be specified multiple times",
    )
    args = parser.parse_args()
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")

    config = Config.load(args.config)
    driver_types = [load_driver_type(spec) for spec in args.driver]
    runtime = MapperRuntime(config, driver_types)
    install_signal_handlers(runtime)
    emit(
        "MAPPER_STARTED",
        {
            "name": config.common.name,
            "version": config.common.version,
            "protocol": config.common.protocol,
            "edgecore_socket": config.common.edgecore_sock,
            "mapper_socket": config.grpc_server.socket_path,
        },
    )
    try:
        runtime.start()
        runtime.wait()
    finally:
        runtime.stop()


if __name__ == "__main__":
    main()

