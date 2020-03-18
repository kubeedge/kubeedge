---
title: Edge Authentication Design
authors:
  - "@ls889"
  - "@GsssC"
  - "@XJangel"
approvers:
  - "@kevin-wangzefeng"
  - "@fisherxu"
  - "@kadisi"

creation-date: 2020-02-06
last-updated: 2020-03-18
status: alpha

---

# Edge Authentication Design

## motivation

The current connection between EdgeCore and CloudCore, that is, the authentication and authorization of the edge node to the cloud requires manual replication of the certificate, which has poor scalability and flexibility. And this root certificate is not within the k8s native authentication and authorization system and requires additional maintenance.

## goal

To implement that edge nodes automatically apply for a certificate when joining the cluster, and k8s API server approves the certificate, then they can  establish mutual authentication TLS with CloudCore.

## design detail

We implement the auto authentication based on the kubeedge installer keadm. The design idea is to reuse the k8s authentication mechanism.

![token.png](../images/edgeAuthentication/edge_authentication.jpg)

##### Each step in the picture is described as follows:

step 0: Once CloudCore starts, CloudCore applies to the API server for `cluster-info` configmap and tokenlist. The `cluster-info` configmap is for edgecore to verify the API server while the tokenlist is to ensure the token of EdgeCore is valid.

**To have EdgeCore trust the API server:**

step 1-2: After running `keadm join ...`, keadm asks the CloudCore for the `cluster-info` configmap. Then the CloudCore responses.

step 3: Then EdgeCore calculates the hashcode of cert to compare with flag discovery-token-ca-cert-hash to verify the API server.

**To have the API server trust the EdgeCore:**

step 4: Edgecore will create CSR if they are same in step3. Then the EdgeCore sends CSR to CloudCore with its token. 

step 5: CloudCore then compare this token with token list. If it is in the token list, CloudCore will create a new client with token and ca.crt to communicate with API server. Next it forward the CSR of EdgeCore to the API server through the client above. If it is not in, CloudCore will ignore this request.

step 6: API server passes the CSR to the component controller manager. The controller manager will approve the CSR.

step 7-8: The CloudCore then forward the certificate to the EdgeCore. 

step 9: The EdgeCore establish mutual authentication TLS with CloudCore.

step 10: The CloudCore communicate with API server through https.

Note: 

Compare between step 4 and step 9:  In step 4, EdgeCore makes a CSR to CloudCore based on **token **while in step 9, EdgeCore communicates with cloudcore through **certificate**.

compare between step 5 and step 10:  In step 5, client is created by token and ca.crt. while in step 10, client is created by original kubeconifg(this kubeconfig has **more permissions** but client with token can only make a CSR).



### keadm init

In this command, we add the step of creating token following the example of kubeadm(token is stored in etcd as a secret). The token is **only** have the rights of making a Certificate Signing Request(CSR) which is used when edge nodes apply to join the cluster.

Note: This token will be expired after 24 hours. Then you can get a new one running `keadm token create`.

And this token only has the rights of applying for a node certificate in k8s cluster.

when CloudCore starts, it gets `cluster info` configmap from API server and calculates a hash code which is named discovery-token-ca-cert-hash and used to join to cluster by EdgeCore.

```shell
curl -k -v -XGET  -H "Accept: application/json, */*" -H "User-Agent: kubeadm/v1.15.3 (linux/amd64) kubernetes/2d3c76f" 'https://<master ip:port>/api/v1/namespaces/kube-public/configmaps/cluster-info'
```

CloudCore also applies to the API server for token list to distinguish the valid token from invalid when edge nodes apply for the certificate.

when the command `keadm init` finished,  It will show as following:

```shell
Then you can join any number of edge nodes by runnning the following on each node:
keadm join --cloudcore-ipport=<ip:ipport address> --edgenode-name=<unique string as identifer> --token=<token id> --discovery-token-ca-cert-hash=<hash of cluster cert>
```

### keadm join

The flags of command `keadm join` should add `--token` and `--discovery-token-ca-cert-hash` which are used to indicate edge identity and verify master identity respectively. So the current `keadm join` should be as follow:

```shell
keadm join --cloudcore-ipport=<ip:ipport address> --edgenode-name=<unique string as identifer> --token=<token id> --discovery-token-ca-cert-hash=<hash of cluster cert>
```



#### 1. discovery cluster-info

**This step is to have the node trust the Kubernetes master.**

Edgecore apply for `cluster-info` configmap from CloudCore. Then EdgeCore calculates the hashcode of cert to compare with flag discovery-token-ca-cert-hash to verify the master.

#### 2. preflight checks

Check version of OS and install subsequently the required pre-requisites using supported steps. Currently we will support **ONLY** (Ubuntu & CentOS)

Check and install all the pre-requisites, which are

- Docker (currently 18.06.0ce3-0~ubuntu) and check if service is up.
- mosquitto (latest available in OS repos) and check if running
- EdgeCore
  - generate necessary config：edge.yaml and modules.yaml
  - generate kubeconfig based on token and ca.crt：bootstrap-edge.conf
  - run EdgeCore



The above two steps are done by keadm. Then next steps are by EdgeCore.

#### 3. TLS bootstrap

**This step is to have API server trust the CloudCore.**

The EdgeCore generates a pair of keys. One of these which is named public key and other information are used to apply for a certificate. The other named private key is stored locally. 

Then EdgeCore sends CSR to CloudCore with its token. 

Cloudcore then compares this token with its token list. If it is in the token list, Cloudcore will create a new client with token and ca.crt to communicate with API server. 

API server passes the CSR to the component controller manager. The controller manager automaticly approves the request so the certificate of edge is OK. The Cloudcore then forward the certificate to the edge. 

From now on, the EdgeCore can establish mutual authentication TLS with Cloudcore.