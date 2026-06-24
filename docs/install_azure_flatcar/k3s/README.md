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
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml 

cd
/home/core/master_prepare.sh
/home/core/master_install.sh
```

Let the cluster spin up and then run `keadm gettoken --kube-config $KUBECONFIG` to obtain the token for edgecore installation on edge VM.

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
10-containerd-net.conflist   master_install.sh  master_uninstall.sh  traefik-ingress-route-tcp0.yaml  traefik-ingress-route-tcp2.yaml  traefik-ingress-route-tcp4.yaml
cloudcore-chart-1.18.tar.gz  master_prepare.sh  traefik-config.yaml  traefik-ingress-route-tcp1.yaml  traefik-ingress-route-tcp3.yaml  values.yaml
```

on edge VM:
`ls /home/core/`
```
10-containerd-net.conflist  edge_install.sh  edge_prepare.sh  edge_uninstall.sh  edgecore.yaml
```


## after cloudcore installation on master

On the master vm perform these commands and compare output to expected.

`kubectl get nodes`
```
NAME          STATUS   ROLES                  AGE     VERSION
kube-master   Ready    control-plane,master   3m36s   v1.28.6+k3s2
```
`kubectl -nkube-system get all -o wide`
```bash
NAME                                          READY   STATUS      RESTARTS   AGE     IP          NODE          NOMINATED NODE   READINESS GATES
pod/local-path-provisioner-84db5d44d9-cb5kq   1/1     Running     0          3m44s   10.42.0.3   kube-master   <none>           <none>
pod/coredns-6799fbcd5-rlhk8                   1/1     Running     0          3m44s   10.42.0.6   kube-master   <none>           <none>
pod/helm-install-traefik-crd-4wzcv            0/1     Completed   0          3m44s   10.42.0.5   kube-master   <none>           <none>
pod/helm-install-traefik-tn44p                0/1     Completed   1          3m37s   10.42.0.7   kube-master   <none>           <none>
pod/svclb-traefik-3a171b25-7krc9              7/7     Running     0          3m29s   10.42.0.8   kube-master   <none>           <none>
pod/traefik-f9d5856dd-f8kk9                   1/1     Running     0          3m29s   10.42.0.9   kube-master   <none>           <none>
pod/metrics-server-67c658944b-f9t8l           1/1     Running     0          3m44s   10.42.0.4   kube-master   <none>           <none>

NAME                     TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                                                                                                      AGE     SELECTOR
service/kube-dns         ClusterIP      10.43.0.10      <none>        53/UDP,53/TCP,9153/TCP                                                                                       3m55s   k8s-app=kube-dns
service/metrics-server   ClusterIP      10.43.124.211   <none>        443/TCP                                                                                                      3m54s   k8s-app=metrics-server
service/traefik          LoadBalancer   10.43.186.68    10.0.0.4      30000:30129/TCP,30001:31595/TCP,30002:30214/TCP,30003:30898/TCP,30004:30338/TCP,80:30795/TCP,443:30585/TCP   3m29s   app.kubernetes.io/instance=traefik-kube-system,app.kubernetes.io/name=traefik

NAME                                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE     CONTAINERS                                                                              IMAGES                                                                                                                                                                                  SELECTOR
daemonset.apps/svclb-traefik-3a171b25   1         1         1       1            1           <none>          3m29s   lb-tcp-30000,lb-tcp-30001,lb-tcp-30002,lb-tcp-30003,lb-tcp-30004,lb-tcp-80,lb-tcp-443   rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5   app=svclb-traefik-3a171b25

NAME                                     READY   UP-TO-DATE   AVAILABLE   AGE     CONTAINERS               IMAGES                                    SELECTOR
deployment.apps/local-path-provisioner   1/1     1            1           3m55s   local-path-provisioner   rancher/local-path-provisioner:v0.0.24    app=local-path-provisioner
deployment.apps/coredns                  1/1     1            1           3m55s   coredns                  rancher/mirrored-coredns-coredns:1.10.1   k8s-app=kube-dns
deployment.apps/traefik                  1/1     1            1           3m29s   traefik                  rancher/mirrored-library-traefik:2.10.5   app.kubernetes.io/instance=traefik-kube-system,app.kubernetes.io/name=traefik
deployment.apps/metrics-server           1/1     1            1           3m55s   metrics-server           rancher/mirrored-metrics-server:v0.6.3    k8s-app=metrics-server

NAME                                                DESIRED   CURRENT   READY   AGE     CONTAINERS               IMAGES                                    SELECTOR
replicaset.apps/local-path-provisioner-84db5d44d9   1         1         1       3m45s   local-path-provisioner   rancher/local-path-provisioner:v0.0.24    app=local-path-provisioner,pod-template-hash=84db5d44d9
replicaset.apps/coredns-6799fbcd5                   1         1         1       3m45s   coredns                  rancher/mirrored-coredns-coredns:1.10.1   k8s-app=kube-dns,pod-template-hash=6799fbcd5
replicaset.apps/traefik-f9d5856dd                   1         1         1       3m30s   traefik                  rancher/mirrored-library-traefik:2.10.5   app.kubernetes.io/instance=traefik-kube-system,app.kubernetes.io/name=traefik,pod-template-hash=f9d5856dd
replicaset.apps/metrics-server-67c658944b           1         1         1       3m45s   metrics-server           rancher/mirrored-metrics-server:v0.6.3    k8s-app=metrics-server,pod-template-hash=67c658944b

NAME                                 COMPLETIONS   DURATION   AGE     CONTAINERS   IMAGES                                      SELECTOR
job.batch/helm-install-traefik-crd   1/1           17s        3m54s   helm         rancher/klipper-helm:v0.8.2-build20230815   batch.kubernetes.io/controller-uid=736c3264-9336-4366-a1d8-37eb6e13555b
job.batch/helm-install-traefik       1/1           11s        3m38s   helm         rancher/klipper-helm:v0.8.2-build20230815   batch.kubernetes.io/controller-uid=a56a98b1-cd2d-4912-9836-e2852c913a4b
```

`kubectl get ns`
```
NAME              STATUS   AGE
kube-system       Active   4m38s
kube-public       Active   4m38s
kube-node-lease   Active   4m38s
default           Active   4m38s
kubeedge          Active   3m7s
```

`kubectl -nkubeedge get all -o wide`
```
NAME                             READY   STATUS    RESTARTS   AGE     IP         NODE          NOMINATED NODE   READINESS GATES
pod/cloudcore-69d64c8b78-9dfkz   1/1     Running   0          3m31s   10.0.0.4   kube-master   <none>           <none>

NAME                TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                                             AGE     SELECTOR
service/cloudcore   ClusterIP   10.43.76.213   <none>        10000/TCP,10001/UDP,10002/TCP,10003/TCP,10004/TCP   3m31s   k8s-app=kubeedge,kubeedge=cloudcore

NAME                                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE     CONTAINERS               IMAGES                     SELECTOR
daemonset.apps/edge-eclipse-mosquitto   0         0         0       0            0           <none>          3m31s   edge-eclipse-mosquitto   eclipse-mosquitto:1.6.15   k8s-app=eclipse-mosquitto,kubeedge=eclipse-mosquitto

NAME                        READY   UP-TO-DATE   AVAILABLE   AGE     CONTAINERS   IMAGES                       SELECTOR
deployment.apps/cloudcore   1/1     1            1           3m31s   cloudcore    kubeedge/cloudcore:v1.15.1   k8s-app=kubeedge,kubeedge=cloudcore

NAME                                   DESIRED   CURRENT   READY   AGE     CONTAINERS   IMAGES                       SELECTOR
replicaset.apps/cloudcore-69d64c8b78   1         1         1       3m31s   cloudcore    kubeedge/cloudcore:v1.15.1   k8s-app=kubeedge,kubeedge=cloudcore,pod-template-hash=69d64c8b78
```


`curl -Ik https://localhost:6443`
```bash
HTTP/2 401 
audit-id: 9b456fae-5913-4f26-b08f-4c972ec882b0
cache-control: no-cache, private
content-type: application/json
content-length: 157
date: Sun, 28 Jul 2024 06:34:47 GMT
```
`curl -Ik https://localhost:10000`
```bash
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```
`curl -Ik https://localhost:10002`
```bash
HTTP/2 404 
content-type: text/plain; charset=utf-8
content-length: 19
date: Sun, 28 Jul 2024 06:35:26 GMT
```

`curl -v -k https://localhost:10002/ca.crt`
```bash
*   Trying [::1]:10002...
* Connected to localhost (::1) port 10002
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
*  start date: Jul 28 06:31:08 2024 GMT
*  expire date: Jul  4 06:31:08 2124 GMT
*  issuer: CN=KubeEdge
*  SSL certificate verify result: unable to get local issuer certificate (20), continuing anyway.
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* using HTTP/2
* [HTTP/2] [1] OPENED stream for https://localhost:10002/ca.crt
* [HTTP/2] [1] [:method: GET]
* [HTTP/2] [1] [:scheme: https]
* [HTTP/2] [1] [:authority: localhost:10002]
* [HTTP/2] [1] [:path: /ca.crt]
* [HTTP/2] [1] [user-agent: curl/8.4.0]
* [HTTP/2] [1] [accept: */*]
> GET /ca.crt HTTP/2
> Host: localhost:10002
> User-Agent: curl/8.4.0
> Accept: */*
> 
< HTTP/2 200 
< content-type: application/octet-stream
< content-length: 380
< date: Sun, 28 Jul 2024 06:36:36 GMT
< 
Warning: Binary output can mess up your terminal. Use "--output -" to tell 
Warning: curl to output it to your terminal anyway, or consider "--output 
Warning: <FILE>" to save to a file.
* Failure writing output to destination
* Connection #0 to host localhost left intact
```

`curl -Ik https://localhost:10003`
```bash
HTTP/2 404 
content-type: text/plain; charset=utf-8
x-content-type-options: nosniff
content-length: 19
date: Sun, 28 Jul 2024 06:35:45 GMT
```
`curl -Ik https://localhost:10004`
```bash
curl: (56) OpenSSL SSL_read: OpenSSL/3.0.9: error:0A000412:SSL routines::sslv3 alert bad certificate, errno 0
```

## after edgecore installation on edge

On the **edge** vm perform these commands and compare output to expected.

`export IP_MASTER=<your master IP>`

`curl -Ik https://$IP_MASTER:6443`
```bash
HTTP/2 401 
audit-id: fd2c4ece-ff45-425c-bb86-86945ed4178b
cache-control: no-cache, private
content-type: application/json
content-length: 157
date: Sun, 28 Jul 2024 06:44:55 GMT
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
*   Trying 172.167.133.228:30002...
* Connected to 172.167.133.228 (172.167.133.228) port 30002
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
*  start date: Jul 28 06:31:08 2024 GMT
*  expire date: Jul  4 06:31:08 2124 GMT
*  issuer: CN=KubeEdge
*  SSL certificate verify result: unable to get local issuer certificate (20), continuing anyway.
* using HTTP/2
* [HTTP/2] [1] OPENED stream for https://172.167.133.228:30002/ca.crt
* [HTTP/2] [1] [:method: GET]
* [HTTP/2] [1] [:scheme: https]
* [HTTP/2] [1] [:authority: 172.167.133.228:30002]
* [HTTP/2] [1] [:path: /ca.crt]
* [HTTP/2] [1] [user-agent: curl/8.4.0]
* [HTTP/2] [1] [accept: */*]
> GET /ca.crt HTTP/2
> Host: 172.167.133.228:30002
> User-Agent: curl/8.4.0
> Accept: */*
> 
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
< HTTP/2 200 
< content-type: application/octet-stream
< content-length: 380
< date: Sun, 28 Jul 2024 06:45:36 GMT
< 
Warning: Binary output can mess up your terminal. Use "--output -" to tell 
Warning: curl to output it to your terminal anyway, or consider "--output 
Warning: <FILE>" to save to a file.
* Failure writing output to destination
* Connection #0 to host 172.167.133.228 left intact
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
97155228cf11a       5dade4ce550b8       3 minutes ago       Running             edge-eclipse-mosquitto   0                   57a13761db4b6       edge-eclipse-mosquitto-9hzqt
```

Optimally all should be running and not in exited mode. [See](#open-issues).

`crictl --runtime-endpoint unix:///run/containerd/containerd.sock images`
```bash
IMAGE                                     TAG                 IMAGE ID            SIZE
docker.io/kubeedge/installation-package   v1.17.0             263653b0572e8       81.3MB
docker.io/library/eclipse-mosquitto       1.6.15              5dade4ce550b8       5.52MB
registry.k8s.io/pause                     3.8                 4873874c08efc       311kB
```

On the **master** vm perform these commands and compare output to expected.
`kubectl get nodes -o wide`
```bash
NAME          STATUS   ROLES                  AGE     VERSION                    INTERNAL-IP   EXTERNAL-IP   OS-IMAGE                                             KERNEL-VERSION   CONTAINER-RUNTIME
kube-master   Ready    control-plane,master   19m     v1.28.6+k3s2               10.0.0.4      <none>        Flatcar Container Linux by Kinvolk 3815.2.5 (Oklo)   6.1.96-flatcar   containerd://1.7.11-k3s2
kube-edge1    Ready    agent,edge             7m47s   v1.28.6-kubeedge-v1.17.0   10.0.0.4      <none>        Flatcar Container Linux by Kinvolk 3815.2.5 (Oklo)   6.1.96-flatcar   containerd://1.7.13
```

We can see the new virtual node `kube-edge1` in state `Ready`.

`kubectl get all --all-namespaces -o wide`
```bash
NAMESPACE     NAME                                          READY   STATUS            RESTARTS   AGE     IP          NODE          NOMINATED NODE   READINESS GATES
kube-system   pod/local-path-provisioner-84db5d44d9-cb5kq   1/1     Running           0          19m     10.42.0.3   kube-master   <none>           <none>
kube-system   pod/coredns-6799fbcd5-rlhk8                   1/1     Running           0          19m     10.42.0.6   kube-master   <none>           <none>
kube-system   pod/helm-install-traefik-crd-4wzcv            0/1     Completed         0          19m     10.42.0.5   kube-master   <none>           <none>
kube-system   pod/helm-install-traefik-tn44p                0/1     Completed         1          19m     10.42.0.7   kube-master   <none>           <none>
kube-system   pod/svclb-traefik-3a171b25-7krc9              7/7     Running           0          19m     10.42.0.8   kube-master   <none>           <none>
kube-system   pod/traefik-f9d5856dd-f8kk9                   1/1     Running           0          19m     10.42.0.9   kube-master   <none>           <none>
kube-system   pod/metrics-server-67c658944b-f9t8l           1/1     Running           0          19m     10.42.0.4   kube-master   <none>           <none>
kubeedge      pod/cloudcore-69d64c8b78-9dfkz                1/1     Running           0          18m     10.0.0.4    kube-master   <none>           <none>
kubeedge      pod/edge-eclipse-mosquitto-9hzqt              1/1     Running           0          8m21s   10.0.0.4    kube-edge1    <none>           <none>
kube-system   pod/svclb-traefik-3a171b25-vgkbl              0/7     SysctlForbidden   0          88s     <none>      kube-edge1    <none>           <none>

NAMESPACE     NAME                     TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                                                                                                      AGE   SELECTOR
default       service/kubernetes       ClusterIP      10.43.0.1       <none>        443/TCP                                                                                                      19m   <none>
kube-system   service/kube-dns         ClusterIP      10.43.0.10      <none>        53/UDP,53/TCP,9153/TCP                                                                                       19m   k8s-app=kube-dns
kube-system   service/metrics-server   ClusterIP      10.43.124.211   <none>        443/TCP                                                                                                      19m   k8s-app=metrics-server
kube-system   service/traefik          LoadBalancer   10.43.186.68    10.0.0.4      30000:30129/TCP,30001:31595/TCP,30002:30214/TCP,30003:30898/TCP,30004:30338/TCP,80:30795/TCP,443:30585/TCP   19m   app.kubernetes.io/instance=traefik-kube-system,app.kubernetes.io/name=traefik
kubeedge      service/cloudcore        ClusterIP      10.43.76.213    <none>        10000/TCP,10001/UDP,10002/TCP,10003/TCP,10004/TCP                                                            18m   k8s-app=kubeedge,kubeedge=cloudcore

NAMESPACE     NAME                                    DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS                                                                              IMAGES                                                                                                                                                                                  SELECTOR
kubeedge      daemonset.apps/edge-eclipse-mosquitto   1         1         1       1            1           <none>          18m   edge-eclipse-mosquitto                                                                  eclipse-mosquitto:1.6.15                                                                                                                                                                k8s-app=eclipse-mosquitto,kubeedge=eclipse-mosquitto
kube-system   daemonset.apps/svclb-traefik-3a171b25   2         2         1       2            1           <none>          19m   lb-tcp-30000,lb-tcp-30001,lb-tcp-30002,lb-tcp-30003,lb-tcp-30004,lb-tcp-80,lb-tcp-443   rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5,rancher/klipper-lb:v0.4.5   app=svclb-traefik-3a171b25

NAMESPACE     NAME                                     READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS               IMAGES                                    SELECTOR
kube-system   deployment.apps/local-path-provisioner   1/1     1            1           19m   local-path-provisioner   rancher/local-path-provisioner:v0.0.24    app=local-path-provisioner
kube-system   deployment.apps/coredns                  1/1     1            1           19m   coredns                  rancher/mirrored-coredns-coredns:1.10.1   k8s-app=kube-dns
kube-system   deployment.apps/traefik                  1/1     1            1           19m   traefik                  rancher/mirrored-library-traefik:2.10.5   app.kubernetes.io/instance=traefik-kube-system,app.kubernetes.io/name=traefik
kube-system   deployment.apps/metrics-server           1/1     1            1           19m   metrics-server           rancher/mirrored-metrics-server:v0.6.3    k8s-app=metrics-server
kubeedge      deployment.apps/cloudcore                1/1     1            1           18m   cloudcore                kubeedge/cloudcore:v1.15.1                k8s-app=kubeedge,kubeedge=cloudcore

NAMESPACE     NAME                                                DESIRED   CURRENT   READY   AGE   CONTAINERS               IMAGES                                    SELECTOR
kube-system   replicaset.apps/local-path-provisioner-84db5d44d9   1         1         1       19m   local-path-provisioner   rancher/local-path-provisioner:v0.0.24    app=local-path-provisioner,pod-template-hash=84db5d44d9
kube-system   replicaset.apps/coredns-6799fbcd5                   1         1         1       19m   coredns                  rancher/mirrored-coredns-coredns:1.10.1   k8s-app=kube-dns,pod-template-hash=6799fbcd5
kube-system   replicaset.apps/traefik-f9d5856dd                   1         1         1       19m   traefik                  rancher/mirrored-library-traefik:2.10.5   app.kubernetes.io/instance=traefik-kube-system,app.kubernetes.io/name=traefik,pod-template-hash=f9d5856dd
kube-system   replicaset.apps/metrics-server-67c658944b           1         1         1       19m   metrics-server           rancher/mirrored-metrics-server:v0.6.3    k8s-app=metrics-server,pod-template-hash=67c658944b
kubeedge      replicaset.apps/cloudcore-69d64c8b78                1         1         1       18m   cloudcore                kubeedge/cloudcore:v1.15.1                k8s-app=kubeedge,kubeedge=cloudcore,pod-template-hash=69d64c8b78

NAMESPACE     NAME                                 COMPLETIONS   DURATION   AGE   CONTAINERS   IMAGES                                      SELECTOR
kube-system   job.batch/helm-install-traefik-crd   1/1           17s        19m   helm         rancher/klipper-helm:v0.8.2-build20230815   batch.kubernetes.io/controller-uid=736c3264-9336-4366-a1d8-37eb6e13555b
kube-system   job.batch/helm-install-traefik       1/1           11s        19m   helm         rancher/klipper-helm:v0.8.2-build20230815   batch.kubernetes.io/controller-uid=a56a98b1-cd2d-4912-9836-e2852c913a4b
```

`kubectl get pods --all-namespaces --field-selector spec.nodeName=kube-edge1 -o wide`
```bash
NAMESPACE     NAME                           READY   STATUS            RESTARTS   AGE   IP         NODE         NOMINATED NODE   READINESS GATES
kubeedge      edge-eclipse-mosquitto-9hzqt   1/1     Running           0          11m   10.0.0.4   kube-edge1   <none>           <none>
kube-system   svclb-traefik-3a171b25-cb8vn   0/7     SysctlForbidden   0          27s   <none>     kube-edge1   <none>           <none>
```

Optimally all pods should be running. Here we observe `pod/svclb-traefik-3a171b25-vgkbl`  as `SysctlForbidden`. This is due to [issue](#open-issues).



# open issues

the edge-node registers but is not ready. Main reason are crashing `kindnet-xxx` pods on both nodes.

## debugging the issue

Debugging the `master` pod:


`kubectl -nkube-system logs pod/svclb-traefik-3a171b25-vgkbl`
```bash
Defaulted container "lb-tcp-30000" out of: lb-tcp-30000, lb-tcp-30001, lb-tcp-30002, lb-tcp-30003, lb-tcp-30004, lb-tcp-80, lb-tcp-443
Error from server: Get "https://10.0.0.4:10351/containerLogs/kube-system/svclb-traefik-3a171b25-vgkbl/lb-tcp-30000": proxy error from 127.0.0.1:6443 while dialing 10.0.0.4:10351, code 502: 502 Bad Gateway
```

`kubectl -nkube-system logs pod/svclb-traefik-3a171b25-7krc9`
```bash
Defaulted container "lb-tcp-30000" out of: lb-tcp-30000, lb-tcp-30001, lb-tcp-30002, lb-tcp-30003, lb-tcp-30004, lb-tcp-80, lb-tcp-443
+ trap exit TERM INT
+ BIN_DIR=/sbin
+ check_iptables_mode
+ set +e
+ lsmod
+ grep -qF nf_tables
+ '[' 0 '=' 0 ]
+ mode=nft
+ set -e
+ info 'nft mode detected'
+ set_nft
+ ln -sf /sbin/xtables-nft-multi /sbin/iptables
[INFO]  nft mode detected
+ ln -sf /sbin/xtables-nft-multi /sbin/iptables-save
+ ln -sf /sbin/xtables-nft-multi /sbin/iptables-restore
+ ln -sf /sbin/xtables-nft-multi /sbin/ip6tables
+ start_proxy
+ echo 0.0.0.0/0
+ grep -Eq :
+ iptables -t filter -I FORWARD -s 0.0.0.0/0 -p TCP --dport 30000 -j ACCEPT
+ echo 10.43.186.68
+ grep -Eq :
+ cat /proc/sys/net/ipv4/ip_forward
+ '[' 1 '==' 1 ]
+ iptables -t filter -A FORWARD -d 10.43.186.68/32 -p TCP --dport 30000 -j DROP
+ iptables -t nat -I PREROUTING -p TCP --dport 30000 -j DNAT --to 10.43.186.68:30000
+ iptables -t nat -I POSTROUTING -d 10.43.186.68/32 -p TCP -j MASQUERADE
+ '[' '!' -e /pause ]
+ mkfifo /pause
```

`kubectl -nkube-system logs pod/traefik-f9d5856dd-f8kk9`
```bash
time="2024-07-28T06:30:02Z" level=info msg="Configuration loaded from flags."
```

`kubectl get networkpolicies -A`
```
No resources found
```

`journalctl -f -u edgecore.service | grep E0728`
```bash
Jul 28 06:59:06 kube-edge1 edgecore[3752]: E0728 06:59:06.707255    3752 process.go:419] metamanager not supported operation: connect
Jul 28 06:59:06 kube-edge1 edgecore[3752]: E0728 06:59:06.713900    3752 cri_stats_provider.go:448] "Failed to get the info of the filesystem with mountpoint" err="unable to find data in memory cache" mountpoint="/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs"
Jul 28 06:59:06 kube-edge1 edgecore[3752]: E0728 06:59:06.713931    3752 kubelet.go:1327] "Image garbage collection failed once. Stats initialization may not have completed yet" err="invalid capacity 0 on image filesystem"
Jul 28 06:59:06 kube-edge1 edgecore[3752]: E0728 06:59:06.733733    3752 kubelet.go:2213] "Skipping pod synchronization" err="[container runtime status check may not have completed yet, PLEG is not healthy: pleg has yet to be successful]"
Jul 28 06:59:06 kube-edge1 edgecore[3752]: E0728 06:59:06.871182    3752 imitator.go:266] failed to unmarshal message content to unstructured obj: Object 'Kind' is missing in '{"metadata":{"name":"kube-edge1","creationTimestamp":null,"labels":{"beta.kubernetes.io/arch":"amd64","beta.kubernetes.io/os":"linux","kubernetes.io/arch":"amd64","kubernetes.io/hostname":"kube-edge1","kubernetes.io/os":"linux","node-role.kubernetes.io/agent":"","node-role.kubernetes.io/edge":""},"annotations":{"volumes.kubernetes.io/controller-managed-attach-detach":"true"}},"spec":{},"status":{"capacity":{"cpu":"4","ephemeral-storage":"27527148Ki","hugepages-1Gi":"0","hugepages-2Mi":"0","memory":"16378896Ki","pods":"110"},"allocatable":{"cpu":"4","ephemeral-storage":"25369019555","hugepages-1Gi":"0","hugepages-2Mi":"0","memory":"16276496Ki","pods":"110"},"conditions":[{"type":"MemoryPressure","status":"False","lastHeartbeatTime":"2024-07-28T06:59:06Z","lastTransitionTime":"2024-07-28T06:59:06Z","reason":"KubeletHasSufficientMemory","message":"kubelet has sufficient memory available"},{"type":"DiskPressure","status":"False","lastHeartbeatTime":"2024-07-28T06:59:06Z","lastTransitionTime":"2024-07-28T06:59:06Z","reason":"KubeletHasNoDiskPressure","message":"kubelet has no disk pressure"},{"type":"PIDPressure","status":"False","lastHeartbeatTime":"2024-07-28T06:59:06Z","lastTransitionTime":"2024-07-28T06:59:06Z","reason":"KubeletHasSufficientPID","message":"kubelet has sufficient PID available"},{"type":"Ready","status":"True","lastHeartbeatTime":"2024-07-28T06:59:06Z","lastTransitionTime":"2024-07-28T06:59:06Z","reason":"EdgeReady","message":"edge is posting ready status"}],"addresses":[{"type":"InternalIP","address":"10.0.0.4"},{"type":"Hostname","address":"kube-edge1"}],"daemonEndpoints":{"kubeletEndpoint":{"Port":10350}},"nodeInfo":{"machineID":"c1e13fd8c0c94a40a93fad849a0ef6f7","systemUUID":"4ee9b459-e216-4602-888c-a9121c230337","bootID":"599023b2-1905-4a41-bf2c-231948d047d1","kernelVersion":"6.1.96-flatcar","osImage":"Flatcar Container Linux by Kinvolk 3815.2.5 (Oklo)","containerRuntimeVersion":"containerd://1.7.13","kubeletVersion":"v1.28.6-kubeedge-v1.17.0","kubeProxyVersion":"v0.0.0-master+$Format:%H$","operatingSystem":"linux","architecture":"amd64"}}}'
Jul 28 06:59:16 kube-edge1 edgecore[3752]: E0728 06:59:16.828778    3752 serviceaccount.go:112] query meta "default"/"kubeedge"/[]string(nil)/3607/v1.BoundObjectReference{Kind:"Pod", APIVersion:"v1", Name:"edge-eclipse-mosquitto-9hzqt", UID:"1b2f95a3-b3af-417c-931d-cf092de77f8f"} length error
Jul 28 07:00:11 kube-edge1 edgecore[3752]: E0728 07:00:11.704121    3752 manager.go:126] get k8s CA failed, send sync message k8s/ca.crt failed: timeout to get response for message 8b25eb05-60a2-4ed3-afb5-a636155ff24f
```


# test on edge deployment

on the master VM: 
run `kubectl apply -f /home/core/nginx-onlyedge.yaml`. Obtain the IP for the below use by running `kubectl get pods --all-namespaces -owide` and looking for pods running on `kube-edge1`.

on the edge VM:
`curl 10.88.0.2` or whatever the IP of the nginx service should result in:
```bash
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
```

`crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps -a` 
```bash
CONTAINER           IMAGE               CREATED              STATE               NAME                     ATTEMPT             POD ID              POD
2444aa701c90d       295c7be079025       About a minute ago   Running             nginx                    0                   5d94c63092e21       nginx
89f5e1f6773bd       5dade4ce550b8       3 minutes ago        Running             edge-eclipse-mosquitto   0                   cdff367e66ec6       edge-eclipse-mosquitto-fr5tj
```

Nginx pod is alive and running !