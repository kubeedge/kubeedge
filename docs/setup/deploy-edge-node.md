# Deploy Edge Node Manually

+ Copy the `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` file and change `metadata.name` to the name of the edge node

    ```shell
    mkdir -p ~/kubeedge/yaml
    cp $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json ~/kubeedge/yaml
    ```

    Node.json

    ```script
    {
      "kind": "Node",
      "apiVersion": "v1",
      "metadata": {
        "name": "edge-node",
        "labels": {
          "name": "edge-node",
          "node-role.kubernetes.io/edge": ""
        }
      }
    }
    ```

**Note:** 
1. the `metadata.name` must keep in line with edgecore's config `modules.edged.hostnameOverride`.

2. Make sure role is set to edge for the node. For this a key of the form `"node-role.kubernetes.io/edge"` must be present in `metadata.labels`.
If role is not set for the node, the pods, configmaps and secrets created/updated in the cloud cannot be synced with the node they are targeted for.

+ Deploy edge node (**you must run the command on cloud side**)

```shell
kubectl apply -f ~/kubeedge/yaml/node.json
```

