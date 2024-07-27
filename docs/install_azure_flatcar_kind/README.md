# host
> **Prerequisites:** azure command line tools, wget, tar installed. For instance use a macbook as host and brew install missing components if necessary. Additionally it is assumed that id_rsa.pub exists under ~/.ssh; else create via ssh-keygen.
> {.is-warning}



Do:
```bash
./az_create_infrastructure.sh --location uksouth
./prepare.sh
```

# master

`ssh core@<ip_master>`


```bash
sudo su
export PATH="$PATH:/opt/bin"
mount --bind /opt/bin /usr/local/bin # flatcar linux /usr is read-only
chmod u+x /home/core/*.sh

cd
/home/core/master_prepare.sh
/home/core/master_install.sh
```

Let the cluster spin up and then run `keadm gettoken` to obtain the token for edgecore installation on edge VM.

If you want to uninstall, run:
`/home/core/master_uninstall.sh`

If you want to then re-install just run `master_install.sh` again.

# edge1

`ssh core@<ip_edge1>`

```bash
sudo su
export PATH="$PATH:/opt/bin"
mount --bind /opt/bin /usr/local/bin # flatcar linux /usr is read-only
chmod u+x /home/core/*.sh

cd
/home/core/edge_prepare.sh
export TOKEN=<your token from above>
/home/core/edge_install.sh --token $TOKEN
```

After a **maximum of 10 seconds** the install should reach `9. Install Complete!`, else something is wrong.

If you want to uninstall, run:
`/home/core/edge_uninstall.sh`

If you want to then re-install just run `edge_install.sh` again.


# check

This section can be used to check if the installation worked out.

## after prepare on host

on master VM:
`ls /home/core/`
```
10-containerd-net.conflist  cloudcore-chart-1.18.tar.gz  kind-config.yaml  master_install.sh  master_prepare.sh  master_uninstall.sh  values.yaml
```

on edge VM:
`ls /home/core/`
```
edge_install.sh  edge_prepare.sh  edge_uninstall.sh  edgecore.yaml
```


## after cloudcore installation on master

On the master vm perform these commands and compare output to expected.

`kind get clusters`

TODO: output here

`kubectl -nkube-system get all -o wide`
```bash
NAME                                                      READY   STATUS    RESTARTS   AGE   IP           NODE                          NOMINATED NODE   READINESS GATES
pod/coredns-5dd5756b68-k4lq2                              1/1     Running   0          48s   10.244.0.4   app-1-cluster-control-plane   <none>           <none>
pod/coredns-5dd5756b68-n46vm                              1/1     Running   0          48s   10.244.0.3   app-1-cluster-control-plane   <none>           <none>
pod/etcd-app-1-cluster-control-plane                      1/1     Running   0          61s   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
pod/kindnet-49vb6                                         1/1     Running   0          48s   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
pod/kube-apiserver-app-1-cluster-control-plane            1/1     Running   0          61s   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
pod/kube-controller-manager-app-1-cluster-control-plane   1/1     Running   0          61s   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
pod/kube-proxy-qlml9                                      1/1     Running   0          48s   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
pod/kube-scheduler-app-1-cluster-control-plane            1/1     Running   0          61s   172.18.0.2   app-1-cluster-control-plane   <none>           <none>

NAME               TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)                  AGE   SELECTOR
service/kube-dns   ClusterIP   10.96.0.10   <none>        53/UDP,53/TCP,9153/TCP   62s   k8s-app=kube-dns

NAME                        DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE   CONTAINERS    IMAGES                                          SELECTOR
daemonset.apps/kindnet      1         1         1       1            1           kubernetes.io/os=linux   60s   kindnet-cni   docker.io/kindest/kindnetd:v20240513-cd2ac642   app=kindnet
daemonset.apps/kube-proxy   1         1         1       1            1           kubernetes.io/os=linux   62s   kube-proxy    registry.k8s.io/kube-proxy:v1.28.9              k8s-app=kube-proxy

NAME                      READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS   IMAGES                                    SELECTOR
deployment.apps/coredns   2/2     2            2           62s   coredns      registry.k8s.io/coredns/coredns:v1.10.1   k8s-app=kube-dns

NAME                                 DESIRED   CURRENT   READY   AGE   CONTAINERS   IMAGES                                    SELECTOR
replicaset.apps/coredns-5dd5756b68   2         2         2       48s   coredns      registry.k8s.io/coredns/coredns:v1.10.1   k8s-app=kube-dns,pod-template-hash=5dd5756b68
```

`kubectl get ns`
```
NAME                 STATUS   AGE
default              Active   6m13s
kube-node-lease      Active   6m13s
kube-public          Active   6m14s
kube-system          Active   6m14s
kubeedge             Active   8s
local-path-storage   Active   6m7s
```

`kubectl -nkubeedge get all -o wide`
```
NAME                             READY   STATUS    RESTARTS   AGE   IP           NODE                          NOMINATED NODE   READINESS GATES
pod/cloudcore-7d59cb6d74-ch9fs   1/1     Running   0          46s   10.244.0.5   app-1-cluster-control-plane   <none>           <none>

NAME                TYPE       CLUSTER-IP     EXTERNAL-IP   PORT(S)                                                                           AGE   SELECTOR
service/cloudcore   NodePort   10.96.184.94   <none>        10000:30000/TCP,10001:30001/UDP,10002:30002/TCP,10003:30003/TCP,10004:30004/TCP   46s   k8s-app=kubeedge,kubeedge=cloudcore

NAME                                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS               IMAGES                     SELECTOR
daemonset.apps/edge-eclipse-mosquitto   0         0         0       0            0           <none>          46s   edge-eclipse-mosquitto   eclipse-mosquitto:1.6.15   k8s-app=eclipse-mosquitto,kubeedge=eclipse-mosquitto

NAME                        READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS   IMAGES                       SELECTOR
deployment.apps/cloudcore   1/1     1            1           46s   cloudcore    kubeedge/cloudcore:v1.15.1   k8s-app=kubeedge,kubeedge=cloudcore

NAME                                   DESIRED   CURRENT   READY   AGE   CONTAINERS   IMAGES                       SELECTOR
replicaset.apps/cloudcore-7d59cb6d74   1         1         1       46s   cloudcore    kubeedge/cloudcore:v1.15.1   k8s-app=kubeedge,kubeedge=cloudcore,pod-template-hash=7d59cb6d74
```

```bash
curl -Ik https://localhost:6443
HTTP/2 403 
audit-id: ae47f274-a3bc-41a0-b771-aeab7177ddb1
cache-control: no-cache, private
content-type: application/json
x-content-type-options: nosniff
x-kubernetes-pf-flowschema-uid: a1b209ae-303c-4b90-afb9-7a3725d43ea9
x-kubernetes-pf-prioritylevel-uid: 905d8fe8-52c1-4f25-afee-a12d1b5ef651
content-length: 218
date: Sat, 27 Jul 2024 11:18:18 GMT
curl -Ik https://localhost:30000
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
curl -Ik https://localhost:30002
HTTP/2 404 
content-type: text/plain; charset=utf-8
content-length: 19
date: Sat, 27 Jul 2024 09:03:23 GMT
curl -Ik https://localhost:30003
HTTP/2 404 
content-type: text/plain; charset=utf-8
x-content-type-options: nosniff
content-length: 19
date: Sat, 27 Jul 2024 09:03:34 GMT
curl -Ik https://localhost:30004
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```

## after edgecore installation on edge

On the **edge** vm perform these commands and compare output to expected.

export IP_MASTER=<your master IP>
```bash
curl -Ik https://$IP_MASTER:6443
HTTP/2 403 
audit-id: dd2ec60e-1ceb-47cf-a0aa-f4d1ac30d312
cache-control: no-cache, private
content-type: application/json
x-content-type-options: nosniff
x-kubernetes-pf-flowschema-uid: a1b209ae-303c-4b90-afb9-7a3725d43ea9
x-kubernetes-pf-prioritylevel-uid: 905d8fe8-52c1-4f25-afee-a12d1b5ef651
content-length: 218
date: Sat, 27 Jul 2024 11:19:00 GMT
curl -Ik https://$IP_MASTER:30000
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
curl -Ik https://$IP_MASTER:30002
HTTP/2 404 
content-type: text/plain; charset=utf-8
content-length: 19
date: Sat, 27 Jul 2024 09:08:54 GMT
curl -Ik https://$IP_MASTER:30003
HTTP/2 404 
content-type: text/plain; charset=utf-8
x-content-type-options: nosniff
content-length: 19
date: Sat, 27 Jul 2024 09:09:05 GMT
curl -Ik https://$IP_MASTER:30004
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```


`crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps -a`
```bash
CONTAINER           IMAGE               CREATED              STATE               NAME                ATTEMPT             POD ID              POD
4e355b7a41ee4       4950bb10b3f87       3 seconds ago        Running             kindnet-cni         2                   df485cf0a17c3       kindnet-5fg66
0103b91a3bccd       4950bb10b3f87       45 seconds ago       Exited              kindnet-cni         1                   df485cf0a17c3       kindnet-5fg66
705250ddde4a1       342a759d88156       About a minute ago   Running             kube-proxy          0                   cca491942936b       kube-proxy-7x6s4
```

Optimally all should be running and not in exited mode. [See](#open-issues).

`crictl --runtime-endpoint unix:///run/containerd/containerd.sock images`
```bash
IMAGE                                     TAG                  IMAGE ID            SIZE
docker.io/kindest/kindnetd                v20240202-8f1494ea   4950bb10b3f87       27.8MB
docker.io/kubeedge/installation-package   v1.17.0              263653b0572e8       81.3MB
registry.k8s.io/kube-proxy                v1.28.6              342a759d88156       26.4MB
registry.k8s.io/pause                     3.8                  4873874c08efc       311kB
```

On the **master** vm perform these commands and compare output to expected.
`kubectl get nodes -o wide`
```bash
NAME                          STATUS     ROLES           AGE     VERSION                    INTERNAL-IP   EXTERNAL-IP   OS-IMAGE                                             KERNEL-VERSION   CONTAINER-RUNTIME
app-1-cluster-control-plane   Ready      control-plane   10m     v1.28.6                    172.18.0.2    <none>        Debian GNU/Linux 12 (bookworm)                       6.1.96-flatcar   containerd://1.7.13
kube-edge1                    NotReady   agent,edge      6m32s   v1.28.6-kubeedge-v1.17.0   10.0.0.4      <none>        Flatcar Container Linux by Kinvolk 3815.2.5 (Oklo)   6.1.96-flatcar   containerd://1.7.13
```

We can see the new virtual node `kube-edge1`. If no errors would occur the `kube-edge1` would be in state `Ready`. But [see](#open-issues).

`kubectl get all --all-namespaces -o wide`
```bash
NAMESPACE            NAME                                                      READY   STATUS    RESTARTS       AGE     IP           NODE                          NOMINATED NODE   READINESS GATES
kube-system          pod/coredns-5dd5756b68-pv7fd                              1/1     Running   0              10m     10.244.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/coredns-5dd5756b68-qlhc8                              1/1     Running   0              10m     10.244.0.3   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/etcd-app-1-cluster-control-plane                      1/1     Running   0              11m     172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kindnet-5fg66                                         1/1     Running   11 (18s ago)   7m10s   10.0.0.4     kube-edge1                    <none>           <none>
kube-system          pod/kindnet-sx48v                                         0/1     Error     6 (3m8s ago)   10m     172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-apiserver-app-1-cluster-control-plane            1/1     Running   0              11m     172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-controller-manager-app-1-cluster-control-plane   1/1     Running   0              11m     172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-proxy-7x6s4                                      1/1     Running   0              7m10s   10.0.0.4     kube-edge1                    <none>           <none>
kube-system          pod/kube-proxy-zx4xk                                      1/1     Running   0              10m     172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-scheduler-app-1-cluster-control-plane            1/1     Running   0              11m     172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kubeedge             pod/cloudcore-7d59cb6d74-thblh                            1/1     Running   0              10m     10.244.0.5   app-1-cluster-control-plane   <none>           <none>
local-path-storage   pod/local-path-provisioner-7577fdbbfb-tccdv               1/1     Running   0              10m     10.244.0.4   app-1-cluster-control-plane   <none>           <none>

NAMESPACE     NAME                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                                                                           AGE   SELECTOR
default       service/kubernetes   ClusterIP   10.96.0.1      <none>        443/TCP                                                                           11m   <none>
kube-system   service/kube-dns     ClusterIP   10.96.0.10     <none>        53/UDP,53/TCP,9153/TCP                                                            11m   k8s-app=kube-dns
kubeedge      service/cloudcore    NodePort    10.96.13.228   <none>        10000:30000/TCP,10001:30001/UDP,10002:30002/TCP,10003:30003/TCP,10004:30004/TCP   11m   k8s-app=kubeedge,kubeedge=cloudcore

NAMESPACE     NAME                                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE   CONTAINERS               IMAGES                                          SELECTOR
kube-system   daemonset.apps/kindnet                  2         2         1       2            1           kubernetes.io/os=linux   11m   kindnet-cni              docker.io/kindest/kindnetd:v20240202-8f1494ea   app=kindnet
kube-system   daemonset.apps/kube-proxy               2         2         2       2            2           kubernetes.io/os=linux   11m   kube-proxy               registry.k8s.io/kube-proxy:v1.28.6              k8s-app=kube-proxy
kubeedge      daemonset.apps/edge-eclipse-mosquitto   0         0         0       0            0           <none>                   11m   edge-eclipse-mosquitto   eclipse-mosquitto:1.6.15                        k8s-app=eclipse-mosquitto,kubeedge=eclipse-mosquitto

NAMESPACE            NAME                                     READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS               IMAGES                                                        SELECTOR
kube-system          deployment.apps/coredns                  2/2     2            2           11m   coredns                  registry.k8s.io/coredns/coredns:v1.10.1                       k8s-app=kube-dns
kubeedge             deployment.apps/cloudcore                1/1     1            1           11m   cloudcore                kubeedge/cloudcore:v1.15.1                                    k8s-app=kubeedge,kubeedge=cloudcore
local-path-storage   deployment.apps/local-path-provisioner   1/1     1            1           11m   local-path-provisioner   docker.io/kindest/local-path-provisioner:v20240202-8f1494ea   app=local-path-provisioner

NAMESPACE            NAME                                                DESIRED   CURRENT   READY   AGE   CONTAINERS               IMAGES                                                        SELECTOR
kube-system          replicaset.apps/coredns-5dd5756b68                  2         2         2       10m   coredns                  registry.k8s.io/coredns/coredns:v1.10.1                       k8s-app=kube-dns,pod-template-hash=5dd5756b68
kubeedge             replicaset.apps/cloudcore-7d59cb6d74                1         1         1       10m   cloudcore                kubeedge/cloudcore:v1.15.1                                    k8s-app=kubeedge,kubeedge=cloudcore,pod-template-hash=7d59cb6d74
local-path-storage   replicaset.apps/local-path-provisioner-7577fdbbfb   1         1         1       10m   local-path-provisioner   docker.io/kindest/local-path-provisioner:v20240202-8f1494ea   app=local-path-provisioner,pod-template-hash=7577fdbbfb
```

Optimally all pods should be running. Here however we see that the `pod/kindnet-xxxx` are constantly crashing. This is due to [issue](#open-issues).

`kubectl get pods --all-namespaces --field-selector spec.nodeName=kube-edge1 -o wide`
```bash
NAMESPACE     NAME               READY   STATUS    RESTARTS       AGE   IP         NODE         NOMINATED NODE   READINESS GATES
kube-system   kindnet-5fg66      1/1     Running   15 (45s ago)   10m   10.0.0.4   kube-edge1   <none>           <none>
kube-system   kube-proxy-7x6s4   1/1     Running   0              10m   10.0.0.4   kube-edge1   <none>           <none>
```


# open issues

the edge-node registers but is not ready. Main reason are crashing `kindnet-xxx` pods on both nodes.

## debugging the issue

Debugging the `master` pod:
`kubectl -nkube-system logs pod/kindnet-sx48v`
```bash
I0727 17:38:03.913887       1 routes.go:62] Adding route {Ifindex: 0 Dst: 10.244.1.0/24 Src: <nil> Gw: 10.0.0.4 Flags: [] Table: 0} 
I0727 17:38:03.913933       1 main.go:204] Failed to reconcile routes, retrying after error: network is unreachable
I0727 17:38:04.914894       1 main.go:223] Handling node with IPs: map[172.18.0.2:{}]
I0727 17:38:04.914950       1 main.go:227] handling current node
I0727 17:38:04.914967       1 main.go:223] Handling node with IPs: map[10.0.0.4:{}]
I0727 17:38:04.914976       1 main.go:250] Node kube-edge1 has CIDR [10.244.1.0/24] 
I0727 17:38:04.915103       1 routes.go:62] Adding route {Ifindex: 0 Dst: 10.244.1.0/24 Src: <nil> Gw: 10.0.0.4 Flags: [] Table: 0} 
I0727 17:38:04.915172       1 main.go:204] Failed to reconcile routes, retrying after error: network is unreachable
I0727 17:38:06.916340       1 main.go:223] Handling node with IPs: map[172.18.0.2:{}]
I0727 17:38:06.916395       1 main.go:227] handling current node
I0727 17:38:06.916412       1 main.go:223] Handling node with IPs: map[10.0.0.4:{}]
I0727 17:38:06.916421       1 main.go:250] Node kube-edge1 has CIDR [10.244.1.0/24] 
I0727 17:38:06.916535       1 routes.go:62] Adding route {Ifindex: 0 Dst: 10.244.1.0/24 Src: <nil> Gw: 10.0.0.4 Flags: [] Table: 0} 
I0727 17:38:06.916605       1 main.go:204] Failed to reconcile routes, retrying after error: network is unreachable
I0727 17:38:09.917606       1 main.go:223] Handling node with IPs: map[172.18.0.2:{}]
I0727 17:38:09.917712       1 main.go:227] handling current node
I0727 17:38:09.917741       1 main.go:223] Handling node with IPs: map[10.0.0.4:{}]
I0727 17:38:09.917757       1 main.go:250] Node kube-edge1 has CIDR [10.244.1.0/24] 
I0727 17:38:09.917959       1 routes.go:62] Adding route {Ifindex: 0 Dst: 10.244.1.0/24 Src: <nil> Gw: 10.0.0.4 Flags: [] Table: 0} 
I0727 17:38:09.918078       1 main.go:204] Failed to reconcile routes, retrying after error: network is unreachable
```
Somehow kind is not able to register a new ip address range for `kube-edge1` node. Why?


Debugging the `edge` pod:
`kubectl -nkube-system logs pod/kindnet-5fg66 `
```bash
Error from server: Get "https://10.0.0.4:10351/containerLogs/kube-system/kindnet-5fg66/kindnet-cni": dial tcp 10.0.0.4:10351: connect: connection refused
```
Shouldn't this work?

`crictl --runtime-endpoint unix:///run/containerd/containerd.sock logs 9fcc183f4772e`
```bash
I0727 17:38:43.532158       1 main.go:316] probe TCP address app-1-cluster-control-plane:6443
W0727 17:38:43.541120       1 main.go:318] DNS problem app-1-cluster-control-plane:6443: lookup app-1-cluster-control-plane on 168.63.129.16:53: no such host
I0727 17:38:43.541173       1 main.go:93] apiserver not reachable, attempt 0 ... retrying
I0727 17:38:43.541179       1 main.go:316] probe TCP address app-1-cluster-control-plane:6443
W0727 17:38:43.546952       1 main.go:318] DNS problem app-1-cluster-control-plane:6443: lookup app-1-cluster-control-plane on 168.63.129.16:53: no such host
I0727 17:38:43.546974       1 main.go:93] apiserver not reachable, attempt 1 ... retrying
I0727 17:38:44.547085       1 main.go:316] probe TCP address app-1-cluster-control-plane:6443
W0727 17:38:44.565227       1 main.go:318] DNS problem app-1-cluster-control-plane:6443: lookup app-1-cluster-control-plane on 168.63.129.16:53: no such host
I0727 17:38:44.565268       1 main.go:93] apiserver not reachable, attempt 2 ... retrying
I0727 17:38:46.566444       1 main.go:316] probe TCP address app-1-cluster-control-plane:6443
W0727 17:38:46.580531       1 main.go:318] DNS problem app-1-cluster-control-plane:6443: lookup app-1-cluster-control-plane on 168.63.129.16:53: no such host
I0727 17:38:46.580557       1 main.go:93] apiserver not reachable, attempt 3 ... retrying
I0727 17:38:49.582761       1 main.go:316] probe TCP address app-1-cluster-control-plane:6443
W0727 17:38:49.588372       1 main.go:318] DNS problem app-1-cluster-control-plane:6443: lookup app-1-cluster-control-plane on 168.63.129.16:53: no such host
I0727 17:38:49.588399       1 main.go:93] apiserver not reachable, attempt 4 ... retrying
```
What is this IP? `168.63.129.16:53`, certainly not from `master` vm.
How to solve this issue?