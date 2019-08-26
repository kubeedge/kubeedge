##  Deploy the admission webhook

This method will guide you to deploy the admission webhook,
so you need to login to the k8s master node (or where else if you can
operate the cluster with `kubectl`).

The manifests and scripts in `github.com/kubeedge/kubeedge/build/admission`
will be used, so place these files to somewhere you can kubectl with.

First, ensure your k8s cluster can pull admission image. If the
image not exist. We can make one, and push to your registry.

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make admissionimage
```

Then, we need to generate the tls certs. It then will create a secret
 if succeeded.

```bash
cd build/admission
./gen-admission-secret.sh
```

Second, we create k8s resources from the manifests in name order. Before
creating, check the content of each manifest to make sure it meets your
environment.

```bash
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```

Last, please use `kubectl get pods -nkubeedge` to check whether the admission run successfully.