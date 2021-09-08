# The HA of CloudCore(deployed in k8s cluster)

Now we support keadm to install HA CloudCore in k8s cluster.


To install HA, we need to create required k8s resource, included `namespace`, `serviceaccount`,
`clusterrole`, `clusterrolebinding`, `configmap` to provide CloudCore configuration,
`delpoyment` to run CloudCore replicas. And besides these resources, we also need to create CRDs,
included `Device`, `DeviceModel`, `ClusterObjectSync`, `ObjectSync`.

All these resources `yaml` are in [01-ha-prepare.yaml](01-ha-prepare.yaml), [02-ha-configmap.yaml.example](02-ha-configmap.yaml.example),
[03-ha-deployment.yaml.example](03-ha-deployment.yaml.example).

We could create these required resources all by ourselves. But now, all these required resources can be created just using the following command:

`./keadm init --cloudcore-run-mode="container" --advertise-address="XXX.XXX.XXX.XXX" --nodeselector="disktype=ssd,label=value"`

After using this command to create resources successfully, we can refer [keepalived](README.md) to configure the keepalived.