# Deploying Cloudcore with Helm

We alse support deploy cloudcore with [helm](https://helm.sh).

## Prepare

### Install Helm

Install helm refer to [doc](https://helm.sh/docs/intro/install/).

### Clone repo

```shell
$ git clone https://github.com/kubeedge/kubeedge.git
```

## Deploy

```shell
$ cd /path/to/kubeedge/repo
$ helm install cloudcore ./helm/cloudcore/
```

output would be like below:
```shell
NAME: cloudcore
LAST DEPLOYED: Tue Jul 14 10:34:04 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
Finish cloudcore deployment.

You can use following command to get cloudcore:
  export CLOUDCORE_IP=$(kubectl get pods --namespace kubeedge -l "app.kubernetes.io/name=cloudcore,app.kubernetes.io/instance=cloudcore" -o jsonpath="{.items[0].status.hostIP}")
```

Then check:
```shell
$ kubectl get pod -n kubeedge
```

The successful output would be like:
```shell
NAME                         READY   STATUS    RESTARTS   AGE
cloudcore-85f794dfcc-mcr67   1/1     Running   0          2m19s
```

Then you can deploy edgecore refer to [doc](./keadm.md#setup-edge-side-kubeedge-worker-node)
