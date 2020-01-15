# Deploy Edge Node Manually

Edge node can be registered automatically. But if you want to deploy edge node manually, here is an example.

+ Generate node's configuration file

```script
cat>./node.json<<EOF
{
  "kind": "Node",
  "apiVersion": "v1",
  "metadata": {
    "name": $nodename,
    "labels": {
      "name": "$nodename",
      "node-role.kubernetes.io/edge": ""
    }
  }
}
EOF
```

**Note:** 
1. the `metadata.name` must keep in line with edgecore's config `modules.edged.hostnameOverride`.

2. Make sure role is set to edge for the node. For this a key of the form `"node-role.kubernetes.io/edge"` must be present in `metadata.labels`.
If role is not set for the node, the pods, configmaps and secrets created/updated in the cloud cannot be synced with the node they are targeted for.

+ Deploy edge node (**you must run the command on cloud side**)

```shell
kubectl apply -f ./node.json
```

