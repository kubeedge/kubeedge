##  Deploy the cloud part into a k8s cluster

This method will guide you to deploy the cloud part into a k8s cluster,
so you need to login to the k8s master node (or where else if you can
operate the cluster with `kubectl`).

The manifests and scripts in `github.com/kubeedge/kubeedge/build/cloud`
will be used, so place these files to somewhere you can kubectl with.

### Prepare cloud image
Ensure your k8s cluster can pull edge controller image. If the
image not exist. We can make one, and push to your registry.

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make cloudimage
```

### Create secret(Optional)
For version lower than 1.3.0, we need to generate the tls certs and create
`06-secret.yaml` based on it.

```bash
cd build/cloud
../tools/certgen.sh buildSecret | tee ./06-secret.yaml
```

### Update config
Based on `08-service.yaml.example`, create your own service `08-service.yaml`,
to expose cloud hub to outside of k8s cluster, so that edge core can
connect to.

Also check the content of each manifest to make sure it meets your environment.

### Create cloud resources
Create k8s resources from the manifests in name order.

```bash
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```
