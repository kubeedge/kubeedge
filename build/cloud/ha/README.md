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
make cloudimage
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
  vrrp_version 2
  vrrp_garp_master_delay 1
  vrrp_garp_master_refresh 1
}

vrrp_script check_haproxy_alive {
  script "/usr/bin/curl -sLk -o /dev/null -w %{response_code} 10.10.102.101:1984" # this is haproxy service port
  timeout 5
  interval 2
  fall 2
  rise 10
}

vrrp_instance VIP_242 {
  state MASTER
  nopreempt
  interface eth0 # based on your host
  virtual_router_id 242
  priority 1
  advert_int 1
  virtual_ipaddress {
    10.10.102.242/24 # VIP
  }
  authentication {
    auth_type PASS
    auth_pass 1111
  }

  track_script {
    check_haproxy_alive
  }
}
```

- backup:

```yaml
! Configuration File for keepalived

global_defs {
  vrrp_version 2
  vrrp_garp_master_delay 1
  vrrp_garp_master_refresh 1
}

vrrp_script check_haproxy_alive {
  script "/usr/bin/curl -sLk -o /dev/null -w %{response_code} 10.10.102.102:1984" # this is haproxy service port
  timeout 5
  interval 2
  fall 2
  rise 10
}

vrrp_instance VIP_242 {
  state BACKUP
  nopreempt
  interface eth0 # based on your host
  virtual_router_id 242
  priority 1
  advert_int 1
  virtual_ipaddress {
    10.10.102.242/24 # VIP
  }
  authentication {
    auth_type PASS
    auth_pass 1111
  }

  track_script {
    check_haproxy_alive
  }
}
```


## haproxy

The `haproxy` configuration, perform the following operations. Enter the node/etc/kubenetes/plugins/lb - config directory, modify haproxy.cfg file, add the following configuration in a file.You can adjust it according to your needs.

**haproxy.cfg:**

- master:

```config
listen http-10000
  mode tcp
  balance source
  timeout client 3600s
  timeout server 3600s
  bind *:10010
  server node1 10.10.102.101:10000 ckeck inter 2000 rise 2 fall 5
  server node2 10.10.102.102:10000 ckeck inter 2000 rise 2 fall 5 backup
  server node3 10.10.102.103:10000 ckeck inter 2000 rise 2 fall 5 backup
  
listen http-10001
  mode tcp
  balance source
  timeout client 3600s
  timeout server 3600s
  bind *:10011
  server node1 10.10.102.101:10001 ckeck inter 2000 rise 2 fall 5
  server node2 10.10.102.102:10001 ckeck inter 2000 rise 2 fall 5 backup
  server node3 10.10.102.103:10001 ckeck inter 2000 rise 2 fall 5 backup

listen http-10002
  mode tcp
  balance source
  timeout client 3600s
  timeout server 3600s
  bind *:10012
  server node1 10.10.102.101:10002 ckeck inter 2000 rise 2 fall 5
  server node2 10.10.102.102:10002 ckeck inter 2000 rise 2 fall 5 backup
  server node3 10.10.102.103:10002 ckeck inter 2000 rise 2 fall 5 backup
  
listen http-10003
  mode tcp
  balance source
  timeout client 3600s
  timeout server 3600s
  bind *:10013
  server node1 10.10.102.101:10003 ckeck inter 2000 rise 2 fall 5
  server node2 10.10.102.102:10003 ckeck inter 2000 rise 2 fall 5 backup
  server node3 10.10.102.103:10003 ckeck inter 2000 rise 2 fall 5 backup
  
listen http-10004
  mode tcp
  balance source
  timeout client 3600s
  timeout server 3600s
  bind *:10014
  server node1 10.10.102.101:10004 ckeck inter 2000 rise 2 fall 5
  server node2 10.10.102.102:10004 ckeck inter 2000 rise 2 fall 5 backup
  server node3 10.10.102.103:10004 ckeck inter 2000 rise 2 fall 5 backup
```
