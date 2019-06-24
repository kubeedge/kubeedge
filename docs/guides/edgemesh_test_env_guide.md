# Edgemesh end to end test guide
## model
![model](../images/edgemesh/model.jpg)
1. a headless service(a service with selector but ClusterIP is None)
2. one or more pods' labels match the headless service's selector
3. so when request a server: ```<service_name>.<service_namespace>.svc.<cluster>```:
    1. get the service's name and namespace from domain name
    2. query the backend pods from metaManager by service's namespace and name
    3. load balance return the real backend container's hostIP and hostPort

## flow from client to server
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

## how to test end to end
- create a headless service(**no need specify port**):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: test-headless
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
  name: test-deployment
  labels:
    app: whatapp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: whatapp
  template:
    metadata:
      labels:
        app: whatapp
    spec:
      nodeSelector:
        name: ${label of the node server run}
      containers:
      - name: whatapp
        image: docker.io/cloudnativelabs/whats-my-ip:latest
        ports:
        - containerPort: 8080
          hostPort: 8080
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
- create client deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: client-deployment
  labels:
    app: whatapp
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
        name: ${label of the node server run}
      containers:
      - name: whatapp
        image: docker.io/cloudnativelabs/whats-my-ip:latest
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
- client request server: exec into client container and then run command: ```curl http://test-headless.default.svc.cluster:8080```, will get the response from server like: ```HOSTNAME:test-app-686c6dbf98-6hrdq IP:10.11.0.4```