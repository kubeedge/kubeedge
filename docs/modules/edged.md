# EdgeD

## Overview

EdgeD is an edge node module which manages the POD lifecycle. With help of the POD specifications provided as JSON from Cloud via MetaManager, it ensures the containers for the POD's (with containers as described in POD spec) are launched over the edge node, are running and healthy.

EdgeD interfaces with MetaManager module to receive all the configuration calls. It responds to MetaManager module with all config results and queries.

Docker container runtime is used for container and image management. It is the default option for EdgeD. It is a configurable option in config.yaml.

Following functionalities are performed by edgeD :-

- POD Launch, Deletion & Modification
- ConfigMap Addition, Deletion & Modification
- Secret Addition, Deletion & Modification
- Probing Containers for Liveliness and Readiness

## POD Launch, Deletion & Modification

EdgeD supports Adding (Launching), Modification and Deletion of POD's. These requests are sent from Cloud, whenever respective "Kubectl" pod commands are execute over its terminal. The specs are defined in yaml format which is similar to that of Kubernetes. Those spec's can also have Volume and Network specs.

If edgeD doesn't have the images in the spec, which shall be used to launch the pod, it will download it using docker runtime.

By modifying the POD spec and issuing the kubectl command for update, pods can be updated.

Execute the below kubectl commands from Cloud console or terminal

To create a POD
```
kubectl create -f <file path>
Example : kubectl create -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

To modify a POD
```
kubectl apply -f <file path>
Example : kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

To delete a POD
```
kubectl delete -f <file path>
Example : kubectl delete -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

## ConfigMap Addition, Deletion & Modification

ConfigMaps allow the user to decouple configuration artifacts from image content to keep containerized applications portable.

For more detailed study and configuration commands please visit **[here](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/)**


## Secret Addition, Deletion & Modification

Secret objects let the user store and manage sensitive information, such as passwords, OAuth tokens, and ssh keys. Putting this information in a secret is safer and more flexible than putting it verbatim in a Pod Lifecycle definition or in a container image

For more detailed study and configuration commands please visit **[here](https://kubernetes.io/docs/concepts/configuration/secret/)**

## Probing Containers for Liveliness and Readiness 

A Probe is a diagnostics performed periodically by the edged on containers. If configured to check the container is alive or not in certain intervals of time, then liveliness probing is configured. To check if the POD is ready to service or not then readiness probe is configured.