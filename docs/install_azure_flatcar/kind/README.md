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
chmod u+x /home/core/*.sh
mount --bind /opt/bin /usr/local/bin # flatcar linux /usr is read-only
export PATH="$PATH:/opt/bin"

cd
/home/core/master_prepare.sh
/home/core/master_install.sh
```

Let the cluster spin up and then run `keadm gettoken` to obtain the token for edgecore installation on edge VM.

If you want to uninstall, run:
`/home/core/master_uninstall.sh`

If you want to then re-install just run `master_install.sh` again.
If you just want to modify the helm chart values of the `cloudcore` component and redeploy, proceed as follows:
```bash
cd kubeedge-release-1.18/manifests/charts/
vi cloudcore/values.yaml
helm upgrade --install cloudcore ./cloudcore --namespace kubeedge --create-namespace -f ./cloudcore/values.yaml
```

# edge1

`ssh core@<ip_edge1>`

```bash
sudo su
chmod u+x /home/core/*.sh
mount --bind /opt/bin /usr/local/bin # flatcar linux /usr is read-only
export PATH="$PATH:/opt/bin"

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
```
app-1-cluster
```

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


`curl -Ik https://localhost:6443`
```bash
HTTP/2 403 
audit-id: ae47f274-a3bc-41a0-b771-aeab7177ddb1
cache-control: no-cache, private
content-type: application/json
x-content-type-options: nosniff
x-kubernetes-pf-flowschema-uid: a1b209ae-303c-4b90-afb9-7a3725d43ea9
x-kubernetes-pf-prioritylevel-uid: 905d8fe8-52c1-4f25-afee-a12d1b5ef651
content-length: 218
date: Sat, 27 Jul 2024 11:18:18 GMT
```
`curl -Ik https://localhost:30000`
```bash
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```
`curl -Ik https://localhost:30002`
```bash
HTTP/2 404 
content-type: text/plain; charset=utf-8
content-length: 19
date: Sat, 27 Jul 2024 09:03:23 GMT
```
`curl -Ik https://localhost:30003`
```bash
HTTP/2 404 
content-type: text/plain; charset=utf-8
x-content-type-options: nosniff
content-length: 19
date: Sat, 27 Jul 2024 09:03:34 GMT
```
`curl -Ik https://localhost:30004`
```bash
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```

## after edgecore installation on edge

On the **edge** vm perform these commands and compare output to expected.

export IP_MASTER=<your master IP>

`curl -Ik https://$IP_MASTER:6443`
```bash
HTTP/2 403 
audit-id: dd2ec60e-1ceb-47cf-a0aa-f4d1ac30d312
cache-control: no-cache, private
content-type: application/json
x-content-type-options: nosniff
x-kubernetes-pf-flowschema-uid: a1b209ae-303c-4b90-afb9-7a3725d43ea9
x-kubernetes-pf-prioritylevel-uid: 905d8fe8-52c1-4f25-afee-a12d1b5ef651
content-length: 218
date: Sat, 27 Jul 2024 11:19:00 GMT
```

`curl -Ik https://$IP_MASTER:30000`
```bash
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```
`curl -Ik https://$IP_MASTER:30002`
```bash
HTTP/2 404 
content-type: text/plain; charset=utf-8
content-length: 19
date: Sat, 27 Jul 2024 09:08:54 GMT
```
`curl -v -k https://$IP_MASTER:30002/ca.crt`
```bash
*   Trying 172.167.149.202:30002...
* Connected to 172.167.149.202 (172.167.149.202) port 30002
* ALPN: curl offers h2,http/1.1
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
* TLSv1.3 (IN), TLS handshake, Request CERT (13):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
* TLSv1.3 (IN), TLS handshake, Finished (20):
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.3 (OUT), TLS handshake, Certificate (11):
* TLSv1.3 (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / TLS_AES_128_GCM_SHA256
* ALPN: server accepted h2
* Server certificate:
*  subject: O=KubeEdge; CN=KubeEdge
*  start date: Jul 27 17:22:28 2024 GMT
*  expire date: Jul  3 17:22:28 2124 GMT
*  issuer: CN=KubeEdge
*  SSL certificate verify result: unable to get local issuer certificate (20), continuing anyway.
* using HTTP/2
* [HTTP/2] [1] OPENED stream for https://172.167.149.202:30002/ca.crt
* [HTTP/2] [1] [:method: GET]
* [HTTP/2] [1] [:scheme: https]
* [HTTP/2] [1] [:authority: 172.167.149.202:30002]
* [HTTP/2] [1] [:path: /ca.crt]
* [HTTP/2] [1] [user-agent: curl/8.4.0]
* [HTTP/2] [1] [accept: */*]
> GET /ca.crt HTTP/2
> Host: 172.167.149.202:30002
> User-Agent: curl/8.4.0
> Accept: */*
> 
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
< HTTP/2 200 
< content-type: application/octet-stream
< content-length: 381
< date: Sat, 27 Jul 2024 19:11:26 GMT
< 
Warning: Binary output can mess up your terminal. Use "--output -" to tell 
Warning: curl to output it to your terminal anyway, or consider "--output 
Warning: <FILE>" to save to a file.
* Failure writing output to destination
* Connection #0 to host 172.167.149.202 left intact
```
`curl -Ik https://$IP_MASTER:30003`
```bash
HTTP/2 404 
content-type: text/plain; charset=utf-8
x-content-type-options: nosniff
content-length: 19
date: Sat, 27 Jul 2024 09:09:05 GMT
```
`curl -Ik https://$IP_MASTER:30004`
```bash
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```


`crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps -a`
```bash
CONTAINER           IMAGE               CREATED             STATE               NAME                     ATTEMPT             POD ID              POD
c6760e44ee1b2       4950bb10b3f87       37 seconds ago      Exited              kindnet-cni              15                  42820d9839d08       kindnet-qdl9f
d86bfe4619a08       342a759d88156       8 minutes ago       Running             kube-proxy               0                   38fa99ce1a3ef       kube-proxy-vxs79
185ade224d9f5       5dade4ce550b8       9 minutes ago       Running             edge-eclipse-mosquitto   0                   3729b356d3af9       edge-eclipse-mosquitto-9f5bp
```

Optimally all should be running and not in exited mode. [See](#open-issues).

`crictl --runtime-endpoint unix:///run/containerd/containerd.sock images`
```bash
IMAGE                                     TAG                  IMAGE ID            SIZE
docker.io/kindest/kindnetd                v20240202-8f1494ea   4950bb10b3f87       27.8MB
docker.io/kubeedge/installation-package   v1.17.0              263653b0572e8       81.3MB
docker.io/library/eclipse-mosquitto       1.6.15               5dade4ce550b8       5.52MB
registry.k8s.io/kube-proxy                v1.28.6              342a759d88156       26.4MB
registry.k8s.io/pause                     3.8                  4873874c08efc       311kB
```
we observe that `kindnetd` has an [issue](#open-issues).

On the **master** vm perform these commands and compare output to expected.
`kubectl get nodes -o wide`
```bash
NAME                          STATUS   ROLES           AGE   VERSION                    INTERNAL-IP   EXTERNAL-IP   OS-IMAGE                                             KERNEL-VERSION   CONTAINER-RUNTIME
app-1-cluster-control-plane   Ready    control-plane   15m   v1.28.6                    172.18.0.2    <none>        Debian GNU/Linux 12 (bookworm)                       6.1.96-flatcar   containerd://1.7.13
kube-edge1                    Ready    agent,edge      11m   v1.28.6-kubeedge-v1.17.0   10.0.0.4      <none>        Flatcar Container Linux by Kinvolk 3815.2.5 (Oklo)   6.1.96-flatcar   containerd://1.7.13
```

We can see the new virtual node `kube-edge1` and it is **Ready**.

`kubectl get all --all-namespaces -o wide`
```bash
NAMESPACE            NAME                                                      READY   STATUS    RESTARTS        AGE   IP           NODE                          NOMINATED NODE   READINESS GATES
kube-system          pod/coredns-5dd5756b68-k595t                              1/1     Running   1 (14m ago)     16m   10.244.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/coredns-5dd5756b68-nxshq                              1/1     Running   1 (14m ago)     16m   10.244.0.4   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/etcd-app-1-cluster-control-plane                      1/1     Running   1 (14m ago)     16m   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kindnet-pb4qz                                         0/1     Error     7 (5m27s ago)   16m   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kindnet-qdl9f                                         1/1     Running   19 (40s ago)    12m   10.0.0.4     kube-edge1                    <none>           <none>
kube-system          pod/kube-apiserver-app-1-cluster-control-plane            1/1     Running   1 (14m ago)     16m   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-controller-manager-app-1-cluster-control-plane   1/1     Running   1 (14m ago)     16m   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-proxy-hxxxl                                      1/1     Running   1 (14m ago)     16m   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kube-system          pod/kube-proxy-vxs79                                      1/1     Running   0               12m   10.0.0.4     kube-edge1                    <none>           <none>
kube-system          pod/kube-scheduler-app-1-cluster-control-plane            1/1     Running   1 (14m ago)     16m   172.18.0.2   app-1-cluster-control-plane   <none>           <none>
kubeedge             pod/cloudcore-7d59cb6d74-fjn46                            1/1     Running   0               14m   10.244.0.5   app-1-cluster-control-plane   <none>           <none>
kubeedge             pod/edge-eclipse-mosquitto-9f5bp                          1/1     Running   0               12m   10.0.0.4     kube-edge1                    <none>           <none>
local-path-storage   pod/local-path-provisioner-7577fdbbfb-vr9tc               1/1     Running   2 (14m ago)     16m   10.244.0.3   app-1-cluster-control-plane   <none>           <none>

NAMESPACE     NAME                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                                                                           AGE   SELECTOR
default       service/kubernetes   ClusterIP   10.96.0.1      <none>        443/TCP                                                                           16m   <none>
kube-system   service/kube-dns     ClusterIP   10.96.0.10     <none>        53/UDP,53/TCP,9153/TCP                                                            16m   k8s-app=kube-dns
kubeedge      service/cloudcore    NodePort    10.96.55.103   <none>        10000:30000/TCP,10001:30001/UDP,10002:30002/TCP,10003:30003/TCP,10004:30004/TCP   14m   k8s-app=kubeedge,kubeedge=cloudcore

NAMESPACE     NAME                                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE   CONTAINERS               IMAGES                                          SELECTOR
kube-system   daemonset.apps/kindnet                  2         2         1       2            1           kubernetes.io/os=linux   16m   kindnet-cni              docker.io/kindest/kindnetd:v20240202-8f1494ea   app=kindnet
kube-system   daemonset.apps/kube-proxy               2         2         2       2            2           kubernetes.io/os=linux   16m   kube-proxy               registry.k8s.io/kube-proxy:v1.28.6              k8s-app=kube-proxy
kubeedge      daemonset.apps/edge-eclipse-mosquitto   1         1         1       1            1           <none>                   14m   edge-eclipse-mosquitto   eclipse-mosquitto:1.6.15                        k8s-app=eclipse-mosquitto,kubeedge=eclipse-mosquitto

NAMESPACE            NAME                                     READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS               IMAGES                                                        SELECTOR
kube-system          deployment.apps/coredns                  2/2     2            2           16m   coredns                  registry.k8s.io/coredns/coredns:v1.10.1                       k8s-app=kube-dns
kubeedge             deployment.apps/cloudcore                1/1     1            1           14m   cloudcore                kubeedge/cloudcore:v1.15.1                                    k8s-app=kubeedge,kubeedge=cloudcore
local-path-storage   deployment.apps/local-path-provisioner   1/1     1            1           16m   local-path-provisioner   docker.io/kindest/local-path-provisioner:v20240202-8f1494ea   app=local-path-provisioner

NAMESPACE            NAME                                                DESIRED   CURRENT   READY   AGE   CONTAINERS               IMAGES                                                        SELECTOR
kube-system          replicaset.apps/coredns-5dd5756b68                  2         2         2       16m   coredns                  registry.k8s.io/coredns/coredns:v1.10.1                       k8s-app=kube-dns,pod-template-hash=5dd5756b68
kubeedge             replicaset.apps/cloudcore-7d59cb6d74                1         1         1       14m   cloudcore                kubeedge/cloudcore:v1.15.1                                    k8s-app=kubeedge,kubeedge=cloudcore,pod-template-hash=7d59cb6d74
local-path-storage   replicaset.apps/local-path-provisioner-7577fdbbfb   1         1         1       16m   local-path-provisioner   docker.io/kindest/local-path-provisioner:v20240202-8f1494ea   app=local-path-provisioner,pod-template-hash=7577fdbbfb
```

Optimally all pods should be running. Here however we see that the `pod/kindnet-xxxx` are constantly crashing. This is due to [issue](#open-issues).

`kubectl get pods --all-namespaces --field-selector spec.nodeName=kube-edge1 -o wide`
```bash
NAMESPACE     NAME                           READY   STATUS    RESTARTS       AGE   IP         NODE         NOMINATED NODE   READINESS GATES
kube-system   kindnet-qdl9f                  1/1     Running   21 (24s ago)   13m   10.0.0.4   kube-edge1   <none>           <none>
kube-system   kube-proxy-vxs79               1/1     Running   0              13m   10.0.0.4   kube-edge1   <none>           <none>
kubeedge      edge-eclipse-mosquitto-9f5bp   1/1     Running   0              13m   10.0.0.4   kube-edge1   <none>           <none>
```


# open issues

the edge-node registers and is ready. But `kindnet-xxx` pods are crashing on both nodes.

## debugging the issue

Debugging the `master` pod:
`kubectl -nkube-system logs pod/kindnet-sx48v`
```bash
I0728 06:00:10.292440       1 main.go:227] handling current node
I0728 06:00:10.292455       1 main.go:223] Handling node with IPs: map[10.0.0.4:{}]
I0728 06:00:10.292461       1 main.go:250] Node kube-edge1 has CIDR [10.244.1.0/24] 
I0728 06:00:10.292602       1 routes.go:62] Adding route {Ifindex: 0 Dst: 10.244.1.0/24 Src: <nil> Gw: 10.0.0.4 Flags: [] Table: 0} 
I0728 06:00:10.292686       1 main.go:204] Failed to reconcile routes, retrying after error: network is unreachable
I0728 06:00:13.293549       1 main.go:223] Handling node with IPs: map[172.18.0.2:{}]
I0728 06:00:13.293585       1 main.go:227] handling current node
I0728 06:00:13.293619       1 main.go:223] Handling node with IPs: map[10.0.0.4:{}]
I0728 06:00:13.293627       1 main.go:250] Node kube-edge1 has CIDR [10.244.1.0/24] 
I0728 06:00:13.293765       1 routes.go:62] Adding route {Ifindex: 0 Dst: 10.244.1.0/24 Src: <nil> Gw: 10.0.0.4 Flags: [] Table: 0} 
I0728 06:00:13.293837       1 main.go:204] Failed to reconcile routes, retrying after error: network is unreachable
panic: Maximum retries reconciling node routes: network is unreachable
```
Somehow kind is not able to register a new ip address range for `kube-edge1` node. Why?

Get the logs of the `kindnet-cni` container within the pod
`kubectl -nkube-system -ckindnet-cni logs pod/kindnet-sx48v`
```
same output as above, since this is a one container pod.
```

Debugging the `edge` pod:
`kubectl -nkube-system logs pod/kindnet-5fg66`
```bash
Error from server: Get "https://10.0.0.4:10351/containerLogs/kube-system/kindnet-5fg66/kindnet-cni": dial tcp 10.0.0.4:10351: connect: connection refused
```
Shouldn't this work?

`crictl --runtime-endpoint unix:///run/containerd/containerd.sock logs 9fcc183f4772e`
```bash
I0728 06:03:51.654073       1 main.go:316] probe TCP address app-1-cluster-control-plane:6443
W0728 06:03:51.660208       1 main.go:318] DNS problem app-1-cluster-control-plane:6443: lookup app-1-cluster-control-plane on 168.63.129.16:53: no such host
I0728 06:03:51.660235       1 main.go:93] apiserver not reachable, attempt 4 ... retrying
I0728 06:03:55.664596       1 main.go:102] connected to apiserver: https://169.254.30.10:10550
I0728 06:03:55.664688       1 main.go:107] hostIP = 10.0.0.4
podIP = 10.0.0.4
I0728 06:03:55.665648       1 main.go:116] setting mtu 1500 for CNI 
I0728 06:03:55.665672       1 main.go:146] kindnetd IP family: "ipv4"
I0728 06:03:55.665720       1 main.go:150] noMask IPv4 subnets: [10.244.0.0/16]
I0728 06:03:55.821287       1 main.go:191] Failed to get nodes, retrying after error: Get "https://169.254.30.10:10550/api/v1/nodes": dial tcp 169.254.30.10:10550: connect: connection refused
I0728 06:03:55.821599       1 main.go:191] Failed to get nodes, retrying after error: Get "https://169.254.30.10:10550/api/v1/nodes": dial tcp 169.254.30.10:10550: connect: connection refused
I0728 06:03:56.823187       1 main.go:191] Failed to get nodes, retrying after error: Get "https://169.254.30.10:10550/api/v1/nodes": dial tcp 169.254.30.10:10550: connect: connection refused
I0728 06:03:58.824863       1 main.go:191] Failed to get nodes, retrying after error: Get "https://169.254.30.10:10550/api/v1/nodes": dial tcp 169.254.30.10:10550: connect: connection refused
I0728 06:04:01.825641       1 main.go:191] Failed to get nodes, retrying after error: Get "https://169.254.30.10:10550/api/v1/nodes": dial tcp 169.254.30.10:10550: connect: connection refused
```
What is this IP? `169.254.30.10:10550`, certainly not from `master` vm.
How to solve this issue?


`journalctl -f -u edgecore.service | grep E0728`
```bash
Jul 28 06:05:41 kube-edge1 edgecore[5771]: E0728 06:05:41.159320    5771 serviceaccount.go:112] query meta "kube-proxy"/"kube-system"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"kube-proxy-vxs79", UID:"c924383b-f372-42e8-b39c-7e4621f580a8"} length error
Jul 28 06:05:42 kube-edge1 edgecore[5771]: E0728 06:05:42.039336    5771 serviceaccount.go:112] query meta "kindnet"/"kube-system"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"kindnet-qdl9f", UID:"6b193603-44a3-46f1-9cab-65e9516ef0f7"} length error
Jul 28 06:05:42 kube-edge1 edgecore[5771]: E0728 06:05:42.190634    5771 serviceaccount.go:112] query meta "default"/"kubeedge"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"edge-eclipse-mosquitto-9f5bp", UID:"d4bdd061-0649-4a7a-9db7-1eccd5966258"} length error
Jul 28 06:06:02 kube-edge1 edgecore[5771]: E0728 06:06:02.921793    5771 pod_workers.go:1300] "Error syncing pod, skipping" err="failed to \"StartContainer\" for \"kindnet-cni\" with CrashLoopBackOff: \"back-off 10s restarting failed container=kindnet-cni pod=kindnet-qdl9f_kube-system(6b193603-44a3-46f1-9cab-65e9516ef0f7)\"" pod="kube-system/kindnet-qdl9f" podUID="6b193603-44a3-46f1-9cab-65e9516ef0f7"
Jul 28 06:06:35 kube-edge1 edgecore[5771]: E0728 06:06:35.794531    5771 manager.go:126] get k8s CA failed, send sync message k8s/ca.crt failed: timeout to get response for message ad3bf161-0840-41e7-850f-fa2b3c665f45
Jul 28 06:06:46 kube-edge1 edgecore[5983]: E0728 06:06:46.048081    5983 process.go:419] metamanager not supported operation: connect
Jul 28 06:06:46 kube-edge1 edgecore[5983]: E0728 06:06:46.053394    5983 cri_stats_provider.go:448] "Failed to get the info of the filesystem with mountpoint" err="unable to find data in memory cache" mountpoint="/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs"
Jul 28 06:06:46 kube-edge1 edgecore[5983]: E0728 06:06:46.053427    5983 kubelet.go:1327] "Image garbage collection failed once. Stats initialization may not have completed yet" err="invalid capacity 0 on image filesystem"
Jul 28 06:06:46 kube-edge1 edgecore[5983]: E0728 06:06:46.070083    5983 kubelet.go:2213] "Skipping pod synchronization" err="[container runtime status check may not have completed yet, PLEG is not healthy: pleg has yet to be successful]"
Jul 28 06:06:46 kube-edge1 edgecore[5983]: E0728 06:06:46.254765    5983 imitator.go:266] failed to unmarshal message content to unstructured obj: Object 'Kind' is missing in '{"metadata":{"name":"kube-edge1","creationTimestamp":null,"labels":{"beta.kubernetes.io/arch":"amd64","beta.kubernetes.io/os":"linux","kubernetes.io/arch":"amd64","kubernetes.io/hostname":"kube-edge1","kubernetes.io/os":"linux","node-role.kubernetes.io/agent":"","node-role.kubernetes.io/edge":""},"annotations":{"volumes.kubernetes.io/controller-managed-attach-detach":"true"}},"spec":{},"status":{"capacity":{"cpu":"4","ephemeral-storage":"27527148Ki","hugepages-1Gi":"0","hugepages-2Mi":"0","memory":"16378896Ki","pods":"110"},"allocatable":{"cpu":"4","ephemeral-storage":"25369019555","hugepages-1Gi":"0","hugepages-2Mi":"0","memory":"16276496Ki","pods":"110"},"conditions":[{"type":"MemoryPressure","status":"False","lastHeartbeatTime":"2024-07-28T06:06:46Z","lastTransitionTime":"2024-07-28T06:06:46Z","reason":"KubeletHasSufficientMemory","message":"kubelet has sufficient memory available"},{"type":"DiskPressure","status":"False","lastHeartbeatTime":"2024-07-28T06:06:46Z","lastTransitionTime":"2024-07-28T06:06:46Z","reason":"KubeletHasNoDiskPressure","message":"kubelet has no disk pressure"},{"type":"PIDPressure","status":"False","lastHeartbeatTime":"2024-07-28T06:06:46Z","lastTransitionTime":"2024-07-28T06:06:46Z","reason":"KubeletHasSufficientPID","message":"kubelet has sufficient PID available"},{"type":"Ready","status":"True","lastHeartbeatTime":"2024-07-28T06:06:46Z","lastTransitionTime":"2024-07-28T06:06:46Z","reason":"EdgeReady","message":"edge is posting ready status"}],"addresses":[{"type":"InternalIP","address":"10.0.0.4"},{"type":"Hostname","address":"kube-edge1"}],"daemonEndpoints":{"kubeletEndpoint":{"Port":10350}},"nodeInfo":{"machineID":"df6437c2c14f4a60a0ec10f27173c503","systemUUID":"bfa8fbff-d5b8-4ed8-a13a-40e8ab98c9b9","bootID":"d845b95a-b079-4315-aa86-8f0e11a2f0ff","kernelVersion":"6.1.96-flatcar","osImage":"Flatcar Container Linux by Kinvolk 3815.2.5 (Oklo)","containerRuntimeVersion":"containerd://1.7.13","kubeletVersion":"v1.28.6-kubeedge-v1.17.0","kubeProxyVersion":"v0.0.0-master+$Format:%H$","operatingSystem":"linux","architecture":"amd64"}}}'
Jul 28 06:06:56 kube-edge1 edgecore[5983]: E0728 06:06:56.043009    5983 cpu_manager.go:395] "RemoveStaleState: removing container" podUID="6b193603-44a3-46f1-9cab-65e9516ef0f7" containerName="kindnet-cni"
Jul 28 06:06:56 kube-edge1 edgecore[5983]: E0728 06:06:56.043023    5983 cpu_manager.go:395] "RemoveStaleState: removing container" podUID="c924383b-f372-42e8-b39c-7e4621f580a8" containerName="kube-proxy"
Jul 28 06:06:56 kube-edge1 edgecore[5983]: E0728 06:06:56.177821    5983 serviceaccount.go:112] query meta "default"/"kubeedge"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"edge-eclipse-mosquitto-9f5bp", UID:"d4bdd061-0649-4a7a-9db7-1eccd5966258"} length error
Jul 28 06:06:56 kube-edge1 edgecore[5983]: E0728 06:06:56.310326    5983 serviceaccount.go:112] query meta "kindnet"/"kube-system"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"kindnet-qdl9f", UID:"6b193603-44a3-46f1-9cab-65e9516ef0f7"} length error
Jul 28 06:06:56 kube-edge1 edgecore[5983]: E0728 06:06:56.390832    5983 serviceaccount.go:112] query meta "kube-proxy"/"kube-system"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"kube-proxy-vxs79", UID:"c924383b-f372-42e8-b39c-7e4621f580a8"} length error
```

