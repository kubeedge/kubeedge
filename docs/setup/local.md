# Deploying Locally

Deploying KubeEdge locally is used to test, never use this way in production environment.

## Limitation

- Need super user rights (or root rights) to run.

## Setup Cloud Side (KubeEdge Master Node)

### Create CRDs

```shell
kubectl apply -f https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/crds/devices/devices_v1alpha2_device.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/crds/devices/devices_v1alpha2_devicemodel.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/crds/reliablesyncs/cluster_objectsync_v1alpha1.yaml
kubectl apply -f https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/crds/reliablesyncs/objectsync_v1alpha1.yaml
```


### Prepare config file

```shell
# cloudcore --minconfig > cloudcore.yaml
```

please refer to [configuration for cloud](../configuration/kubeedge.md#configuration-cloud-side-kubeedge-master) for details.

### Run

```shell
# cloudcore --config cloudcore.yaml
```

Run `cloudcore -h` to get help info and add options if needed.


## Setup Edge Side (KubeEdge Worker Node)

### Prepare config file

- generate config file

```shell
# edgecore --minconfig > edgecore.yaml
```

- get token value at cloud side:

```shell
# kubectl get secret -nkubeedge tokensecret -o=jsonpath='{.data.tokendata}' | base64 -d
```

- update token value in edgecore config file:

```shell
# sed -i -e "s|token: .*|token: ${token}|g" edgecore.yaml
```

The `token` is what above step get.

please refer to [configuration for edge](../configuration/kubeedge.md#configuration-edge-side-kubeedge-worker-node) for details.

### Run

If you want to run cloudcore and edgecore at the same host, run following command first:

```shell
# export CHECK_EDGECORE_ENVIRONMENT="false"
```

Start edgecore:

```shell
# edgecore --config edgecore.yaml
```

Run `edgecore -h` to get help info and add options if needed.
