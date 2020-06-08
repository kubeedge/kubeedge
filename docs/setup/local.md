# Deploying Locally

Deploying KubeEdge locally is used to test, never use this way in production environment.

## Limitation

- Need super user rights (or root rights) to run.

## Setup Cloud Side (KubeEdge Master Node)

### Prepare config file

```shell
# cloudcore --minconfig > cloudcore.yaml
```

Update any fields if needed.

### Run

```shell
# cloudcore --config cloudcore.yaml
```

## Setup Edge Side (KubeEdge Worker Node)

### Prepare config file

```shell
# edgecore --minconfig > edgecore.yaml
```

Update any fields if needed.

### Run

```shell
# edgecore --config edgecore.yaml
```
