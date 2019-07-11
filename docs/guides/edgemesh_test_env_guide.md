# Prerequisites
- install containerd
- ensure that have add follow config items to section ```edged``` in edge.yaml:
```
edged:
    runtime-type: remote
    remote-runtime-endpoint: /var/run/containerd/containerd.sock
    remote-image-endpoint: /var/run/containerd/containerd.sock
    runtime-request-timeout: 2
    podsandbox-image: k8s.gcr.io/pause
```
- install CNI plugin

# Edgemesh test env config guide

## Install Containerd 
+ please refer to the following link to install  
> <https://kubernetes.io/docs/setup/production-environment/container-runtimes/#containerd>

## Install CNI plugin 
+ get cni plugin source code of version (0.2.0) from the github.com. Compile and install.
> It is recommended to use version 0.2.0, which have tested to ensure stability and availability.
> make sure to run on the Go development environment.
```bash
# download the cni-plugin-0.2.0
$ wget https://github.com/containernetworking/plugins/archive/v0.2.0.tar.gz
# Extract the tarball
$ tar -zxvf v0.2.0.tar.gz
# Compile the source code
$ cd ./plugins-0.2.0
$ ./build
# install the plugin after './build'
$ mkdir -p /opt/cni/bin
$ cp ./bin/* /opt/cni/bin/
```
+ configure cni plugin
 
```bash
$ mkdir -p /etc/cni/net.d/
```
   > * please Make sure docker0 does not exist !!
   > * field "bridge" must be "docker0"
   > * field "isGateway" must be true 
   
```bash
$ cat >/etc/cni/net.d/10-mynet.conf <<EOF
{
	"cniVersion": "0.2.0",
	"name": "mynet",
	"type": "bridge",
	"bridge": "docker0",
	"isGateway": true,
	"ipMasq": true,
	"ipam": {
		"type": "host-local",
		"subnet": "10.22.0.0/16",
		"routes": [
			{ "dst": "0.0.0.0/0" }
		]
	}
}
EOF
```

## Configure port mapping manually on node on which server is running
> can see the examples in the next section. 
* step1. execute iptables command as follows 
```bash
$ iptables -t nat -N PORT-MAP
$ iptables -t nat -A PORT-MAP -i docker0 -j RETURN
$ iptables -t nat -A PREROUTING -p tcp -m addrtype --dst-type LOCAL -j PORT-MAP
$ iptables -t nat -A OUTPUT ! -d 127.0.0.0/8 -p tcp -m addrtype --dst-type LOCAL -j PORT-MAP
$ iptables -P FORWARD ACCEPT
```
* step2. execute iptables command as follows 
   > * **portIN** is the service map at the host
   > * **containerIP** is the IP in the container. Can be find out on master by **kubectl get pod -o wide**
   > * **portOUT** is the port that monitored In-container 
```bash
$ iptables -t nat -A PORT-MAP ! -i docker0 -p tcp -m tcp --dport portIN -j DNAT --to-destination containerIP:portOUT
``` 
> + by the way, If you redeployed the service,you can use the command as follows to delete the rule, and perform the second step again.
   ```bash
    $ iptables -t nat -D PORT-MAP 2
  ``` 
## Example for Edgemesh test env
![edgemesh test env example](../images/edgemesh/edgemesh-test-env-example.png)

## Edgemesh end to end test guide
### Model
![model](../images/edgemesh/model.jpg)
1. a headless service(a service with selector but ClusterIP is None)
2. one or more pods' labels match the headless service's selector
3. so when request a server: ```<service_name>.<service_namespace>.svc.<cluster>:<port>```:
    1. get the service's name and namespace from domain name
    2. query the backend pods from metaManager by service's namespace and name
    3. load balance return the real backend container's hostIP and hostPort

### Flow from client to server
![flow](../images/edgemesh/endtoend-test-flow.jpg)
1. client request to server's domain name
2. DNS request hijacked to edgemesh by iptables, return a fake ip
3. request hijacked to edgemesh by iptables
4. edgemesh resolve request, get domain name, protocol, request and so on
5. edgemesh load balance:
    1. get the service name and namespace from the domain name
    2. query backend pod of the service from metaManager
    3. choose a backend based on strategy
6. edgemesh transport request to server wait server response and then response to client

### How to test end to end
- create a headless service(**no need specify port**):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: edgemesh-example-service
  namespace: default
spec:
  clusterIP: None
  selector:
    app: whatapp
```
- create server deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: server
  labels:
    app: whatapp
spec:
  replicas: 2
  selector:
    matchLabels:
      app: whatapp
  template:
    metadata:
      labels:
        app: whatapp
    spec:
      nodeSelector:
        name: edge-node
      containers:
      - name: whatapp
        image: docker.io/cloudnativelabs/whats-my-ip:latest
        ports:
        - containerPort: 8080
          hostPort: 8080
```
- create client deployment:
**replace the image please**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: client
  labels:
    app: client
spec:
  replicas: 1
  selector:
    matchLabels:
      app: client
  template:
    metadata:
      labels:
        app: client
    spec:
      nodeSelector:
        name: edge-node
      containers:
      - name: client
        image: ${your client image for test}
      initContainers:
      - args:
        - -p
        - "8080"
        - -i
        - "192.168.1.2/24,156.43.2.1/26"
        - -t
        - "12345,5432,8080"
        - -c
        - "9292"
        name: init1
        image: docker.io/ytsobd/edgemesh_init:v1.0
        securityContext:
          privileged: true
```
**note: -p: whitelist, only port in whitelist can go out from client to edgemesh then to server**
- client request server: exec into client container and then run command: 
```bash
curl http://edgemesh-example-service.default.svc.cluster:8080
```
 will get the response from server like: ```HOSTNAME:server-5c5868b79f-j4td7 IP:10.22.0.14```
> * **there is two ways to exec the 'curl' command to access your service**
> * 1st: use 'ctr' command attach in the container and make sure there is 'curl' command in the container
    
```bash
$ ctr -n k8s.io c ls
$ ctr -n k8s.io t exec --exec-id 123 <containerID> sh
# if you get error: curl: (6) Could not resolve host: edgemesh-example-service.default.svc.cluster; Unknown error
# please check the /etc/resolv.conf has a right config like: nameserver 8.8.8.8
``` 
> * 2nd: switch the network namespace.(Recommended Use)  
```bash
# first step get the id,this command will return a id start with 'cni-xx'. and make sure the 'xx' is related to the pod which you can get from 'kubectl describe <podName>' 
$ ip netns
# and the use this id to switch the net namespace. And the you can exec curl to access the service
$ ip netns exec <id> bash
``` 
