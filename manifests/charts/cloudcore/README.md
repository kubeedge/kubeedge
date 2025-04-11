# CloudCore Application

Visit https://github.com/kubeedge/kubeedge for more information.

## Install examples

```
helm upgrade --install cloudcore ./cloudcore --namespace kubeedge --create-namespace -f ./cloudcore/values.yaml --set cloudCore.modules.cloudHub.advertiseAddress[0]=192.168.88.6
```

> Note that the parameter `cloudCore.modules.cloudHub.advertiseAddress` is indispensable to start the KubeEdge cloudcore component on the cloud side.

> Add `--dry-run` if only for testing purposes.

## Custom Values

### cloudcore

- `cloudCore.modules.cloudHub.advertiseAddress`, defines the unmissable public IPs which can be accessed by edge nodes.
- `cloudCore.modules.cloudHub.dnsNames`, defines the domain names which can be accessed by edge nodes.
- `cloudCore.hostNetWork`, default `true`, which shares the host network, used for setting the forward iptables rules on the host.
- `cloudCore.image.repository`, default `kubeedge`, defines the image repo.
- `cloudCore.image.tag`, default `v1.12.0`, defines the image tag.
- `cloudCore.image.pullPolicy`, default `IfNotPresent`, defines the policies to pull images.
- `cloudCore.image.pullSecrets`, defines the secrets to pull images.
- `cloudCore.labels`, defines the labels.
- `cloudCore.annotations`, defines the annotations.
- `cloudCore.affinity`, `cloudCore.nodeSelector`, `cloudCore.tolerations`, defines the node scheduling policies.
- `cloudCore.resources`, defines the resources limits and requests.
- `cloudCore.modules.cloudHub.nodeLimit`, defines the edge nodes limits.
- `cloudCore.modules.cloudHub.websocket.enable`, default `true`.
- `cloudCore.modules.cloudHub.quic.enable`, default `false`.
- `cloudCore.modules.cloudHub.https.enable`, default `true`.
- `cloudCore.modules.cloudStream.enable`, default `true`.
- `cloudCore.modules.dynamicController.enable`,  default `false`.
- `cloudCore.modules.router.enable`,  default `false`.
- `cloudCore.service.type`,  default `NodePort`.
- `cloudCore.service.cloudhubNodePort`,  default `30000`, which defines the exposed node port for cloudhub service.
- `cloudCore.service.cloudhubQuicNodePort`,  default `30001`, which defines the exposed node port for cloudhub quic protocol.
- `cloudCore.service.cloudhubHttpsNodePort`,  default `30002`, which defines the exposed node port for cloudhub https protocol.
- `cloudCore.service.cloudstreamNodePort`,  default `30003`, which defines the exposed node port for cloud stream service.
- `cloudCore.service.tunnelNodePort`,  default `30004`, which defines the exposed node port for cloud tunnel service.
- `cloudCore.service.annotations`, defines the annotations for service.

### iptables-manager
- `iptablesManager.enable`,  default `true`
- `iptablesManager.mode`,  default `external`, can be modified to `internal`. The external mode will set up a iptables manager component which shares the host network.
- `iptablesManager.framework`,  default `legacy`, can be modified to `nft`. Iptables of different frameworks are not compatible in the same network environment and need to be configured according to the operating environment.
- `iptablesManager.image.repository`, default `kubeedge`, defines the image repo.
- `iptablesManager.image.tag`, default `v1.19.0`, defines the image tag.
- `iptablesManager.image.pullPolicy`, default `IfNotPresent`, defines the policies to pull images.
- `iptablesManager.image.pullSecrets`, defines the secrets to pull images.
- `iptablesManager.labels`, defines the labels.
- `iptablesManager.annotations`, defines the annotations.
- `iptablesManager.affinity`, `iptablesManager.nodeSelector`, `iptablesManager.tolerations`, defines the node scheduling policies.
- `iptablesManager.resources`, defines the resources limits and requests.

## Uninstall

```
helm uninstall cloudcore -n kubeedge
kubectl delete ns kubeedge
```