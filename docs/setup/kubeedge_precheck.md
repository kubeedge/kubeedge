# KubeEdge Pre-Check

## Status Check

After the Cloud and Edge parts have started, you can use below command to check the edge node status.

On cloud host run,

```shell
kubectl get nodes
````

or

```shell
kubectl get nodes -l node-role.kubernetes.io/edge=
```

Please make sure the `status` of edge node you created is **ready**.

## Deploy Application on cloud side

Try out a sample application deployment by following below steps.

```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
deployment.apps/nginx-deployment created
```

**Note:** Currently, for applications running on edge nodes, we don't support `kubectl logs` and `kubectl exec` commands(will support in future release), support pod to pod communication running on **edge nodes in same subnet** using edgemesh.

Then you can use below command to check if the application is normally running.

Check the pod is up and is `running` state

```shell
kubectl get pods
NAME                               READY   STATUS    RESTARTS   AGE
nginx-deployment-d86dfb797-scfzz   1/1     Running   0          44s
```

Check the deployment is up and is in `running` state

```shell
kubectl get deployments

NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   1/1     1            1           63s
```

### Monitoring containers status

If the container runtime configured to manage containers is containerd , then the following commands can be used to inspect container status and list images.

```shell
sudo ctr --namespace k8s.io containers ls
sudo ctr --namespace k8s.io images ls
sudo crictl exec -ti <containerid> /bin/bash
```

## Run Tests

### Run Edge Unit Tests

 ```shell
 make test
 ```

 To run unit tests of a package individually.

 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge
 cd <path to package to be tested>
 go test -v
 ```

### Run Edge Integration Tests

```shell
make integrationtest
```

### Run E2E Tests

```shell
make e2e
```

### Details and use cases of integration test framework

Please find the [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) to use cases of integration test framework for KubeEdge.
