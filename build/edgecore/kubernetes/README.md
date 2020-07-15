##  Deploy the edge part into a k8s cluster

This method will guide you to deploy the edge part into a k8s cluster,
so you need to login to the k8s master node (or where else if you can
operate the cluster with `kubectl`).

The manifests and scripts in `github.com/kubeedge/kubeedge/build/edgecore/kubernetes`
will be used, so place these files to somewhere you can kubectl with.

First, ensure your k8s cluster can pull edge core image. If the
image not exist. We can make one, and push to your registry.

- Check the container runtime environment

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge/build/edgecore
./run_daemon.sh prepare
```

- Build edge core image

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make edgeimage
```

We create k8s resources from the manifests in name order. Before
creating, check the content of each manifest to make sure it meets your
environment.

Firstly you need to copy edge certs including `edge.crt` and `edge.key` into the folder
`/etc/kubeedge/certs/` on the k8s nodes where you want to deploy edge part.

On the other side, you need to replace `0.0.0.0:10000` with your kubeedge cloud web socket url.
* [url](03-configmap-edgenodeconf.yaml#L20)

The default edge node name is `edgenode1`,
if you want to change edge node name or add new edge node,
you need to replace the following places with your new edge node name.

* [name in 02-edgenode.yaml](02-edgenode.yaml#L4)
* [url in 03-configmap-edgenodeconf.yaml](03-configmap-edgenodeconf.yaml#L20)
* [node-id in 03-configmap-edgenodeconf.yaml](03-configmap-edgenodeconf.yaml#L33)
* [hostname-override in 03-configmap-edgenodeconf.yaml](03-configmap-edgenodeconf.yaml#L36)
* [name in 04-deployment-edgenode.yaml](04-deployment-edgenode.yaml#L4)

```bash
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```
