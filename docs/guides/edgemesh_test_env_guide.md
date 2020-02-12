# Edgemesh test env config guide
## Docker support
* Refer to [Usage](./../setup/installer_setup.md) to prepare KubeEdge environment, make sure "docker0" exists
* Then follow [EdgeMesh guide](#EdgeMesh test guide) to deploy Service

## Usage of EdgeMesh
![edgemesh test env example](../images/edgemesh/edgemesh-test-env-example.png)

## EdgeMesh test guide
### Model
![model](../images/edgemesh/model.jpg)
1. a headless service (a service with selector but ClusterIP is None)
2. one or more pods' labels match the headless service's selector
3. to request a server, use: ```<service_name>.<service_namespace>.svc.<cluster>:<port>```:
    1. get the service's name and namespace from domain name
    2. query all the backend pods from MetaManager by service's namespace and name
    3. LoadBalance returns the real backend containers' hostIP and hostPort

### Flow from client to server
![flow](../images/edgemesh/endtoend-test-flow.jpg)
1. client requests to server by server's domain name
2. DNS being hijacked to EdgeMesh by iptables rules, then a fake ip returned
3. request hijacked to EdgeMesh by iptables rules
4. EdgeMesh resolves request, gets domain name, protocol, request and so on
5. EdgeMesh load balances:
    1. get the service's name and namespace from the domain name
    2. query backend pods of the service from MetaManager
    3. choose a backend based on strategy
6. EdgeMesh transports request to server, wait for server's response and then sends response back to client

### How to test EdgeMesh
Assume we have two edge nodes in ready state, we call them edge node "a" and "b"
```bash
$ kubectl get deployments
NAME          STATUS     ROLES    AGE   VERSION
edge-node-a   Ready      edge     25m   v1.15.3-kubeedge-v1.1.0-beta.0.358+0b7ac7172442b5-dirty
edge-node-b   Ready      edge     25m   v1.15.3-kubeedge-v1.1.0-beta.0.358+0b7ac7172442b5-dirty
master        NotReady   master   8d    v1.15.0
```
Deploy a sample pod from Cloud VM (you may already did it)

**https://github.com/kubeedge/kubeedge/blob/master/build/deployment.yaml**

Copy the deployment.yaml from the above link in cloud host, and run

```bash
$ kubectl create -f deployment.yaml
deployment.apps/nginx-deployment created
``` 
Check the pod is up and is running state, as we could see the pod is running on edge node b

```bash
$ kubectl get pods -o wide
NAME                                READY   STATUS    RESTARTS   AGE   IP           NODE          NOMINATED NODE   READINESS GATES
nginx-deployment-54bf9847f8-sxk94   1/1     Running   0          14m   172.17.0.2   edge-node-b   <none>           <none>
```

Check the deployment is up and is running state
```bash
$ kubectl get deployments
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   1/1     1            1           63s
```

Now create a service for the sample deployment
```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
  namespace: default
spec:
  clusterIP: None
  selector:
    app: nginx
  ports:
    - name: http-0
      port: 12345
      protocol: TCP
      targetPort: 80
```
>* For L4/L7 proxy, specify what protocol a port would use by the port's "name". First HTTP port should be named "http-0" and the second one should be called "http-1", etc.
>* Currently we only support HTTP1.x, more protocols like HTTPS and gRPC coming later

Check the service and endpoints
```bash
$ kubectl get service
NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)     AGE
nginx-svc    ClusterIP   None         <none>        12345/TCP   77m
```
```bash
$ kubectl get endpoints
NAME         ENDPOINTS            AGE
nginx-svc    172.17.0.2:80        81m
```

To request a server, use url like this: ```<service_name>.<service_namespace>.svc.<cluster>:<port>```

In our case, from edge node a or b, run the command:
```bash
$ curl http://nginx-svc.default.svc.cluster.local:12345
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
>* EdgeMesh supports both Host Networking and Container Networking
>* If you ever used EdgeMesh of old version, check your iptables rules. It might affect your test result.  