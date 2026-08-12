"""The built-in PrintDriver is used when no --driver is supplied.

Run from the project root:

    python -m mapper_framework --config config.yaml

Any cloud desired value is printed as CLOUD_DESIRED_VALUE and no hardware is
accessed. This file is intentionally small so it can be copied as a starting
point for a real mapper.
"""

from mapper_framework import PrintDriver


class DemoPrintDriver(PrintDriver):
    protocol_name = "python-demo"

