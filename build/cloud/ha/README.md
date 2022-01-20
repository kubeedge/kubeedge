# The HA of CloudCore(deployed in k8s cluster)

**Note:**

There are several ways to achieve the HA of cloudcore, for example, ingress, keepalived etc. Here we adopt the keepalived. The HA of cloudcore according to ingress will be achieved later.

## Determine the virtual IP of CloudCore

Determine a VIP that the CloudCore service exposed to the edge nodes. Here we recommend `keepalived` to do that. You had better directly schedule pods to specific number of nodes by `nodeSelector` when using  `keepalived`. And you have  to install `keepalived` in each of nodes where CloudCore runs. The configuration of `keepalived` is shown in the end. Here suppose the VIP is 10.10.102.242.

The use of `nodeSelector` is as follow:

```bash
kubectl label nodes [nodename] [key]=[value]  # label the nodes where the cloudcore will run
```

modify the term of `nodeselector`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudcore
spec:
  template:
    spec:
      nodeSelector: # configure the nodeSelector here!
        [key]: [value]
```

## Create k8s resources

The manifests and scripts in `github.com/kubeedge/kubeedge/build/cloud/ha` will be used, so place these files to somewhere you can kubectl with (You have to make some modifications to manifests/scrips to suit your environment.)

First, ensure your k8s cluster can pull cloudcore image. If the image not exist. We can make one, and push to your registry.

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make image WHAT=cloudcore
```

We create k8s resources from the manifests in name order. Before creating, **check the content of each manifest to make sure it meets your environment.**

**Note:** Now the follow manifests don't support `kubectl logs` command yet. If need, you have to make more configuration manually.

### 02-ha-configmap.yaml

Configure the VIP address of CloudCore which is exposed to the edge nodes in the `advertiseAddress`, which will be added to SANs in cert of CloudCore. For example:

```yaml
modules:
  cloudHub:
    advertiseAddress:
    - 10.10.102.242
```

**Note:** If you want to reset the CloudCore, run this before creating k8s resources:

```bash
kubectl delete namespace kubeedge
```

Then create k8s resources:

```shell
cd build/cloud/ha
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```

## keepalived

The `keepalived` configuration we recommend is as following. You can adjust it according to your needs.

**keepalived.conf:**

- master:

```yaml
! Configuration File for keepalived

global_defs {
  router_id lb01
  vrrp_mcast_group4 224.0.0.19
}
# CloudCore
vrrp_script CloudCore_check {
  script "/etc/keepalived/check_cloudcore.sh" # the script for health check
  interval 2
  weight 2
  fall 2
  rise 2
}
vrrp_instance CloudCore {
  state MASTER
  interface eth0 # based on your host
  virtual_router_id 167
  priority 100
  advert_int 1
  authentication {
    auth_type PASS
    auth_pass 1111
  }
  virtual_ipaddress {
    10.10.102.242/24 # VIP
  }
  track_script {
    CloudCore_check
 }
}
```

- backup:

```yaml
! Configuration File for keepalived

global_defs {
  router_id lb02
  vrrp_mcast_group4 224.0.0.19
}
# CloudCore
vrrp_script CloudCore_check {
  script "/etc/keepalived/check_cloudcore.sh" # the script for health check
  interval 2
  weight 2
  fall 2
  rise 2
}
vrrp_instance CloudCore {
  state BACKUP
  interface eth0 # based on your host
  virtual_router_id 167
  priority 99
  advert_int 1
  authentication {
    auth_type PASS
    auth_pass 1111
  }
  virtual_ipaddress {
    10.10.102.242/24 # VIP
  }
  track_script {
    CloudCore_check
 }
}
```

check_cloudcore.sh:

```shell
#!/usr/bin/env bash
http_code=`curl -k -o /dev/null -s -w %{http_code} https://127.0.0.1:10002/readyz`
if [ $http_code == 200 ]; then
    exit 0
else
    exit 1
fi

```

