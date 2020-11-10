# Deploying using Keadm

Keadm is used to install the cloud and edge components of KubeEdge. It is not responsible for installing K8s and runtime, so check dependences section in this [doc](../getting-started.md) first.

Please refer [kubernetes-compatibility](https://github.com/kubeedge/kubeedge#kubernetes-compatibility) to get **Kubernetes compatibility** and determine what version of Kubernetes would be installed.

## Limitation

- Currently support of `keadm` is available for Ubuntu and CentOS OS. RaspberryPi supports is in-progress.
- Need super user rights (or root rights) to run.

## Setup Cloud Side (KubeEdge Master Node)

By default ports `10000` and `10002` in your cloudcore needs to be accessible for your edge nodes.

**Note**: port `10002` only needed since 1.3 release.

`keadm init` will install cloudcore, generate the certs and install the CRDs. It also provides a flag by which a specific version can be set.

**IMPORTANT NOTE:**
1. At least one of kubeconfig or master must be configured correctly, so that it can be used to verify the version and other info of the k8s cluster.
1. Please make sure edge node can connect cloud node using local IP of cloud node, or you need to specify public IP of cloud node with `--advertise-address` flag.
1. `--advertise-address`(only work since 1.3 release) is the address exposed by the cloud side (will be added to the SANs of the CloudCore certificate), the default value is the local IP.

Example:

```shell
# keadm init --advertise-address="THE-EXPOSED-IP"(only work since 1.3 release)
```

Output:
```
Kubernetes version verification passed, KubeEdge installation will start...
...
KubeEdge cloudcore is running, For logs visit:  /var/log/kubeedge/cloudcore.log
```

## (**Only Needed in Pre 1.3 Release**) Manually copy certs.tgz from cloud host to edge host(s)

**Note**: Since release 1.3, feature `EdgeNode auto TLS Bootstrapping` has been added and there is no need to manually copy certificate.

Now users still need to copy the certs to the edge nodes. In the future, it will support the use of tokens for authentication.

On edge host:

```
mkdir -p /etc/kubeedge
```

On cloud host:

```
cd /etc/kubeedge/
scp -r certs.tgz username@edge_node_ip:/etc/kubeedge
```

On edge host untar the certs.tgz file

```
cd /etc/kubeedge
tar -xvzf certs.tgz
```

## Setup Edge Side (KubeEdge Worker Node)

### Get Token From Cloud Side

Run `keadm gettoken` in **cloud side** will return the token, which will be used when joining edge nodes.

```shell
# keadm gettoken
27a37ef16159f7d3be8fae95d588b79b3adaaf92727b72659eb89758c66ffda2.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTAyMTYwNzd9.JBj8LLYWXwbbvHKffJBpPd5CyxqapRQYDIXtFZErgYE
```

### Join Edge Node

`keadm join` will install edgecore and mqtt. It also provides a flag by which a specific version can be set.

Example:

```shell
# keadm join --cloudcore-ipport=192.168.20.50:10000 --token=27a37ef16159f7d3be8fae95d588b79b3adaaf92727b72659eb89758c66ffda2.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTAyMTYwNzd9.JBj8LLYWXwbbvHKffJBpPd5CyxqapRQYDIXtFZErgYE
```

**IMPORTANT NOTE:**
1. `--cloudcore-ipport` flag is a mandatory flag.
1. If you want to apply certificate for edge node automatically, `--token` is needed.
1. The kubeEdge version used in cloud and edge side should be same.

Output:

```shell
Host has mosquit+ already installed and running. Hence skipping the installation steps !!!
...
KubeEdge edgecore is running, For logs visit:  /var/log/kubeedge/edgecore.log
```

### Enable `kubectl logs` Feature

Before metrics-server deployed, `kubectl logs` feature must be activated:

1. Make sure you can find the kubernetes `ca.crt` and `ca.key` files. If you set up your kubernetes cluster by `kubeadm` , those files will be in `/etc/kubernetes/pki/` dir.

    ``` shell
    ls /etc/kubernetes/pki/
    ```

2. Set `CLOUDCOREIPS` env. The environment variable is set to specify the IP address of cloudcore, or a VIP if you have a highly available cluster.

    ```bash
    export CLOUDCOREIPS="192.168.0.139"
    ```
    (Warning: the same **terminal** is essential to continue the work, or it is necessary to type this command again.) Checking the environment variable with the following command:
    ``` shell
    echo $CLOUDCOREIPS
    ```

3. Generate the certificates for **CloudStream** on cloud node, however, the generation file is not in the `/etc/kubeedge/`, we need to copy it from the repository which was git cloned from GitHub.
   Change user to root:
    ```shell
    sudo su
    ```
    Copy certificates generation file from original cloned repository:
    ```shell
    cp $GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh /etc/kubeedge/
    ```
    Change directory to the kubeedge directory:
    ```shell
    cd /etc/kubeedge/
    ```
    Generate certificates from **certgen.sh**
    ```bash
    /etc/kubeedge/certgen.sh stream
    ```

4. It is needed to set iptables on the host. (This command should be executed on every apiserver deployed node.)(In this case, this the master node, and execute this command by root.)
   Run the following command on the host on which each apiserver runs:

    **Note:** You need to set the cloudcoreips variable first

    ```bash
    iptables -t nat -A OUTPUT -p tcp --dport 10350 -j DNAT --to $CLOUDCOREIPS:10003
    ```
    > Port 10003 and 10350 are the default ports for the CloudStream and edgecore,
      use your own ports if you have changed them.

    If you are not sure if you have setting of iptables, and you want to clean all of them.
    (If you set up iptables wrongly, it will block you out of your `kubectl logs` feature)
    The following command can be used to clean up iptables:
    ``` shell
    iptables -F && iptables -t nat -F && iptables -t mangle -F && iptables -X
    ```

5. Modify **both** `/etc/kubeedge/config/cloudcore.yaml` and `/etc/kubeedge/config/edgecore.yaml` on cloudcore and edgecore. Set up **cloudStream** and **edgeStream** to `enable: true`. Change the server IP to the cloudcore IP (the same as $CLOUDCOREIPS).

    Open the YAML file in cloudcore:
    ```shell
    sudo nano /etc/kubeedge/config/cloudcore.yaml
    ```

    Modify the file in the following part (`enable: true`):
    ```yaml
    cloudStream:
      enable: true
      streamPort: 10003
      tlsStreamCAFile: /etc/kubeedge/ca/streamCA.crt
      tlsStreamCertFile: /etc/kubeedge/certs/stream.crt
      tlsStreamPrivateKeyFile: /etc/kubeedge/certs/stream.key
      tlsTunnelCAFile: /etc/kubeedge/ca/rootCA.crt
      tlsTunnelCertFile: /etc/kubeedge/certs/server.crt
      tlsTunnelPrivateKeyFile: /etc/kubeedge/certs/server.key
      tunnelPort: 10004
    ```

    Open the YAML file in edgecore:
    ``` shell
    sudo nano /etc/kubeedge/config/edgecore.yaml
    ```
    Modify the file in the following part (`enable: true`), (`server: 192.168.0.193:10004`):
    ``` yaml
    edgeStream:
      enable: true
      handshakeTimeout: 30
      readDeadline: 15
      server: 192.168.0.139:10004
      tlsTunnelCAFile: /etc/kubeedge/ca/rootCA.crt
      tlsTunnelCertFile: /etc/kubeedge/certs/server.crt
      tlsTunnelPrivateKeyFile: /etc/kubeedge/certs/server.key
      writeDeadline: 15
    ```

6. Restart all the cloudcore and edgecore.

    ``` shell
    sudo su
    ```
    cloudCore:
    ``` shell
    pkill cloudcore
    nohup cloudcore > cloudcore.log 2>&1 &
    ```
    edgeCore:
    ``` shell
    systemctl restart edgecore.service
    ```
    If you fail to restart edgecore, check if that is because of `kube-proxy` and kill it.  **kubeedge** reject it by default, we use a succedaneum called [edgemesh](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/edgemesh-design.md)

    **Note:** the importance is to avoid `kube-proxy` being deployed on edgenode. There are two methods to solve it:

    1. Add the following settings by calling `kubectl edit daemonsets.apps -n kube-system kube-proxy`:
    ``` yaml
    affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: node-role.kubernetes.io/edge
                    operator: DoesNotExist
    ```

    2. If you still want to run `kube-proxy`, ask **edgecore** not to check the environment by adding the env variable in `edgecore.service` :

        ``` shell
        sudo vi /etc/kubeedge/edgecore.service
        ```

        - Add the following line into the **edgecore.service** file:

        ``` shell
        Environment="CHECK_EDGECORE_ENVIRONMENT=false"
        ```

         - The final file should look like this:

        ```
        Description=edgecore.service

        [Service]
        Type=simple
        ExecStart=/root/cmd/ke/edgecore --logtostderr=false --log-file=/root/cmd/ke/edgecore.log
        Environment="CHECK_EDGECORE_ENVIRONMENT=false"

        [Install]
        WantedBy=multi-user.target
        ```

### Support Metrics-server in Cloud
1. The realization of this function point reuses cloudstream and edgestream modules. So you also need to perform all steps of *Enable `kubectl logs` Feature*.

2. Since the kubelet ports of edge nodes and cloud nodes are not the same, the current release version of metrics-server(0.3.x) does not support automatic port identification (It is the 0.4.0 feature), so you need to manually compile the image from master branch yourself now.

    Git clone latest metrics server repository:

    ```bash
    git clone https://github.com/kubernetes-sigs/metrics-server.git
    ```

    Go to the metrics server directory:

    ```bash
    cd metrics-server
    ```

    Make the docker image:

    ```bash
    make container
    ```

    Check if you have this docker image:

    ```bash
    docker images
    ```

    |                  REPOSITORY                           |                    TAG                   |   IMAGE ID   |     CREATE     |  SIZE  |
    |-------------------------------------------------------|------------------------------------------|--------------|----------------|--------|
    | gcr.io/k8s-staging-metrics-serer/ metrics-serer-amd64 | 6d92704c5a68cd29a7a81bce68e6c2230c7a6912 | a24f71249d69 | 19 seconds ago | 57.2MB |
    | metrics-server-kubeedge                               |                 latest                   | aef0fa7a834c | 28 seconds ago | 57.2MB |


    Make sure you change the tag of image by using its IMAGE ID to be compactable with image name in yaml file.

    ```bash
    docker tag a24f71249d69 metrics-server-kubeedge:latest
    ```

3. Apply the deployment yaml. For specific deployment documents, you can refer to https://github.com/kubernetes-sigs/metrics-server/tree/master/manifests.

    **Note:** those iptables below must be applyed on the machine (to be exactly network namespace, so metrics-server needs to run in hostnetwork mode also) metric-server runs on.
    ```
    iptables -t nat -A OUTPUT -p tcp --dport 10350 -j DNAT --to $CLOUDCOREIPS:10003
    ```
    (To direct the request for metric-data from edgecore:10250 through tunnel between cloudcore and edgecore, the iptables is vitally important.)

    Before you deploy metrics-server, you have to make sure that you deploy it on the node which has apiserver deployed on. In this case, that is the master node. As a consequence, it is needed to make master node schedulable by the following command:

    ``` shell
    kubectl taint nodes --all node-role.kubernetes.io/master-
    ```

    Then, in the deployment.yaml file, it must be specified that metrics-server is deployed on master node.
    (The hostname is chosen as the marked label.)
    In **metrics-server-deployment.yaml**
    ``` yaml
        spec:
          affinity:
            nodeAffinity:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms:
                - matchExpressions:
                  #Specify which label in [kubectl get nodes --show-labels] you want to match
                  - key: kubernetes.io/hostname
                    operator: In
                    values:
                    #Specify the value in key
                    - charlie-latest
    ```

**IMPORTANT NOTE:**
1. Metrics-server needs to use hostnetwork network mode.

2. Use the image compiled by yourself and set imagePullPolicy to Never.

3. Enable the feature of --kubelet-use-node-status-port for Metrics-server

    Those settings need to be written in deployment yaml (metrics-server-deployment.yaml) file like this:

    ``` yaml
          volumes:
          # mount in tmp so we can safely use from-scratch images and/or read-only containers
          - name: tmp-dir
            emptyDir: {}
          hostNetwork: true                          #Add this line to enable hostnetwork mode
          containers:
          - name: metrics-server
            image: metrics-server-kubeedge:latest    #Make sure that the REPOSITORY and TAG are correct
            # Modified args to include --kubelet-insecure-tls for Docker Desktop (don't use this flag with a real k8s cluster!!)
            imagePullPolicy: Never                   #Make sure that the deployment uses the image you built up
            args:
              - --cert-dir=/tmp
              - --secure-port=4443
              - --v=2
              - --kubelet-insecure-tls
              - --kubelet-preferred-address-types=InternalDNS,InternalIP,ExternalIP,Hostname
              - --kubelet-use-node-status-port       #Enable the feature of --kubelet-use-node-status-port for Metrics-server
            ports:
            - name: main-port
              containerPort: 4443
              protocol: TCP
    ```

## Reset KubeEdge Master and Worker nodes

### Master
`keadm reset` will stop `cloudcore` and delete KubeEdge related resources from Kubernetes master like `kubeedge` namespace. It doesn't uninstall/remove any of the pre-requisites.

It provides a flag for users to specify kubeconfig path, the default path is `/root/.kube/config`.

 Example:

```shell
 # keadm reset --kube-config=$HOME/.kube/config
```

 ### Node
`keadm reset` will stop `edgecore` and it doesn't uninstall/remove any of the pre-requisites.
