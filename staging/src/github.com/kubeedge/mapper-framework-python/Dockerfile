FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY pyproject.toml ./
COPY mapper_framework ./mapper_framework
COPY config.yaml ./config.yaml

ENTRYPOINT ["python", "-m", "mapper_framework", "--config", "/app/config.yaml"]

