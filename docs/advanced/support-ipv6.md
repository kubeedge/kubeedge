# KubeEdge IPv6 Support

KubeEdge supports IPv6 for CloudHub-to-EdgeHub communication, allowing edge nodes to connect to the cloud control plane using IPv6 addresses. This guide describes the current level of support, supported scenarios, and configuration steps.

## Current Level of Support

KubeEdge provides IPv6 support for the communication channel between CloudCore and EdgeCore. This includes:
- Webhook, CloudHub, and other cloud-side services supporting IPv6.
- EdgeCore joining and communicating with CloudCore via IPv6.
- Edge nodes reporting dual-stack (IPv4/IPv6) addresses.

## Supported Scenarios

- **Dual-Stack Clusters**: KubeEdge integrates with Kubernetes clusters that have IPv4/IPv6 dual-stack networking enabled.
- **HostNetwork Mode**: When CloudCore is configured in `hostNetwork` mode, it can leverage the node's IPv6 address directly.
- **IPv6-Only Joining**: Edge nodes can join the cluster using an IPv6 address for the `cloudcore-ipport`.
- **Dual-IP Reporting**: Edge nodes can report both IPv4 and IPv6 addresses to the cloud, which then appear in the node's status.

## Configuration Guidance

### Cloud Side (Kubernetes Cluster)

1.  **Enable Dual-Stack in K8s**:
    - **kube-apiserver**: Add `--service-cluster-ip-range=<IPv4 CIDR>,<IPv6 CIDR>`.
    - **kube-controller-manager**: Add `--cluster-cidr` and `--service-cluster-ip-range` with both CIDRs.
    - **kube-proxy**: Update the ConfigMap with `clusterCIDR: <IPv4 CIDR>,<IPv6 CIDR>`.
    - **kubelet**: Set `--node-ip=<IPv4 IP>,<IPv6 IP>`.

2.  **Network Plugin (e.g., Calico)**:
    - Configure the IPAM to assign both IPv4 and IPv6.
    - Enable IPv6 support in the plugin's configuration (e.g., Calico DaemonSet environment variables).

3.  **CloudCore Service**:
    Edit the `cloudcore` service to support dual-stack:
    ```yaml
    spec:
      ipFamilies: [IPv4, IPv6]
      ipFamilyPolicy: PreferDualStack
    ```

4.  **Certificates**:
    Ensure the IPv6 address is included in the `advertiseAddress` of the CloudCore configuration. If adding IPv6 post-installation, update the ConfigMap and regenerate the secrets:
    ```bash
    kubectl delete secret -n kubeedge cloudcoresecret casecret
    ```
    Restart CloudCore to regenerate certificates with the new IPv6 address.

### Edge Side

1.  **Joining with IPv6**:
    When using `keadm` to join an edge node, use brackets for the IPv6 address:
    ```bash
    keadm join --cloudcore-ipport=[<IPv6 IP>]:<Port> --token=<Token>
    ```

2.  **Reporting Dual-Stack IPs**:
    Modify `/etc/kubeedge/config/edgecore.yaml` to specify both IPv4 and IPv6 addresses for reporting:
    ```yaml
    modules:
      edged:
        nodeIP: <Node IPv4 IP>,<Node IPv6 IP>
    ```

## Known Limitations & Caveats

- **Manual Reporting Configuration**: By default, edge nodes only report their IPv4 address. Manual modification of `edgecore.yaml` is required for dual-stack reporting.
- **CNI Plugin Support**: Different CNI plugins have varying levels of dual-stack support and configuration requirements.
- **Certificate Regeneration**: If the IPv6 address is not included in the `advertiseAddress` initially, certificates must be manually deleted and regenerated.
- **Address Formatting**: Ensure IPv6 addresses are properly bracketed (e.g., `[::1]:10000`) when used in configuration fields that include a port.
