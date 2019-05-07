---
title: Decentralized Edge Service Identity Framework Design
version: v0.1 (alpha)
authors:
    - "@trilokgm"
approvers:
  - "@qizha"
  - "@CindyXing"
  - "@Baoqiang-Zhang"
  - "@m1093782566"
creation-date: 2019-05-07
last-updated: 2019-05-07
status: pending
---

# Decentralized Edge Service Identity Framework Design

* [Decentralized Edge Service Identity Framework Design](#Decentralized Edge Service Identity Framework Design)
  * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non\-goals](#non-goals)
  * [Proposal](#proposal)
    * [Use Cases](#use-cases)
  * [Design Details](#design-details)  
    * [Node registration and deletion](#nodes-registration-and-deletion)
    * [Application registration and deletion](#application-registration-and-deletion)
  * [Offline Scenarios](#offline-scenarios)
  * [Scalability](#scalability)
  * [Open questions](#open-questions)

## Motivation

Security is a paramount requirement for edge computing architecture as security breaches can make a complete organization to come to a halt (IIot) , data breach can lead to privacy issues and also control of the complete edge computing infrastructure. Each computation process and communication is required to be secure and auditable which requires a strong identity framework which supports assignment of identities for every workload and device, attestation of identities, rotation of secrets, blocking invalid identities from communication channels and information that helps better auditing. Traditional methods of network isolation strategies pose difficulty with scalabiliy and mobility of user devices and applications as network policies and user-specific edge node security management requires manual intervention from administrators and are prone to human errors.

In the context of edge computing, there is a need for decentralized security framework, which works independently of centralized server clusters. Such framework enables all edge computing framework to be able to authenticate and authorize services without a need to communicate to centralized server. A partial dependency of centralized server is still required for administration configuration roles which are required to check who is permitted to change the edge security framework policies.

### Goals (in security-version 0.1)
* Ability to assign identities for edge nodes and workloads.
* Ability to attest the identities assigned.
* Ability to block requests from workloads which are not allowed to communicate.
* Ability to rotate secrets for KubeEdge components after a configured TTL.
* Ability to rotate secrets for user workloads after a configured TTL.
* Ability to rotate secrets for user workloads to obtain secrets without connecting to cloud during the configured TTL.

### Non-goals (in security-version 0.1)

* To issue identities for devices connecting to KubeEdge using mqtt.
* To address secure deployment of certificates for identity servers.
* Sync up node-name or node identities from k8s master to security identities.
* Communication policies.
* Bootstrapping SPIRE components.
* Force revocation of ca bundle and leaf certificates.
* Policy management.
* Spire server federation.

## Proposal
* Use SPIFFE specification based SPIRE for implementation of identity servers.

### SPIFFE and SPIRE
The Secure Production Identity Framework For Everyone (SPIFFE) Project defines a framework and set of standards for identifying and securing communications between services. SPIFFE enables enterprises to transform their security posture from just protecting the edge to consistently securing all inter-service communications deep within their applications. SPIFFE recommends industry standards like TLS and JWT for forming a identity document for every service. The service-level identity AuthN and AuthZ removes the dependency of complex network-level ACL strategies.

More information about SPIFFE can be found at https://github.com/spiffe/spiffe.

SPIRE (SPIFFE Runtime Environment) is a reference implementation of SPIFFE specification. SPIRE manages identities for node and workloads. It provides API for controlling attestation policies and identity issuance and rotation strategies.

More information about SPIRE can be found at https://github.com/spiffe/spire.

### Benefits
Node attestation: Only verifiable edge nodes can join the edge clusters. Every node is issued an identity on verification. In case of failed node attestations, no identity documents can be issued for services running on the node.

Workload attestation: Only verifiable workload can run on edge nodes. In case of failed workload attestations, there are no identities issues for the workloads. All communications are blocked from unverified workloads.

Certificate rotation: Short-lived certificates are generated and rotation policies can be configured for every service communication. There is no need for custom agents and reliance on specific orchestrators for certificate rotation configuration and management.

Automated non-root CA certificate heirarchical deployments: Edge spire servers can be configured to not share any root CA chain for downstream nodes and workloads.

### Why not Kubernetes secrets?

* Kubernetes secrets work in a centralized model of secret distribution. To support offline scenarios, usage of long-lived certificates are used. Usage of long-lived certificates expose keys and certificates for longer duration which will enable theft and therefore are a higher value to an attacker since a stolen key enables the attacker to impersonate the service.
* Known risks of using Kubernetes secrets (here)[https://kubernetes.io/docs/concepts/configuration/secret/#risks]. The presented solution mitigates risks related to protection of secret by application, secret visibility to other instance of application, impersonation of application instance (though, this requires a stronger method of workload selector used for attestation) and accidental storage of secretes on shared volumes by storing in-memory. In-memory storage have issues but pose a harder problem. A better notification framework will be proposed in the forthcoming versions.

## Architecture goal.

* High-level architecture that is achieved in the presented design.\

<img src="../images/security/SpireEdgev0.1-SpireEdgeIntegrationDetailed.png">

* For reference, a PoC without integration with KubeEdge is present in examples.

https://github.com/kubeedge/examples/blob/master/security-demo/README.md

Demo video can be found in the following link:

https://www.youtube.com/watch?v=Nq9EzTrWTRM&t=24s

### Use Cases

* Perform create/delete operations on node entries.
  * Users will create and delete node entries with identity framework by adding nodes and application instances either using kubernetes master or spire cli along with manual deployment of application.
  * Identity framework will perform node attestation based on the entry and attestation plugin configured.

* Perform CRUD operations on workload entries.
   * Users can create, update and delete application instance.
   * Identity framework will perform workload attestation and provide application leaf certificates.

* Workload pod liveness/readiness probe report can be used to understand the connectivity.

* Workload implements workload api to retrieve and rotate certificates.

## Design Details

### Node registration and deletion

Node regisrations are created using exiting kubernetes API. Alternatively, node registrations can also be created using spire-server cli.

<img src="../images/security/SpireEdgev0.1-SpireEdgeIntegrationDetailed.png">

On addition of a new edge node, identity controller watches nodes and automatically creates entry for edge node and edge components with attestation information.

For cloud node registration,  

+ Master node is created once the kubernetes control plane is initialized. 

+ Based on the node type, master or slave, respective edge component are assigned a spiffe id with attestation information as configured.

+ Identity controller creates a new entry with spire server.

+ Cloud spire agent performs node attestation and receives node certificates. The node certificates are used to connect to spire server through node api for workload certificate requests.

+ Cloud hub implements spire workload api to retrieve certificates. Cloud spire agent performs workload attestation. On successful attestation, certificate signing request is made to cloud spire server. On response from cloud spire server, cloud spire agent caches the certificate and responds to cloud hub with private key and certificate.

<img src="../images/security/SpireEdgev0.1-CreateCloudNodeSequence.png">

For edge node registration, 

+ Edge node entry is created at kubernetes api server using kubectl command. 

+ Based on the node type, master or slave, respective edge component are assigned a spiffe id with attestation information as configured.

+ Identity controller creates a new entry with spire server.

+ Cloud spire agent performs node attestation and receives node certificates. The node certificates are used to connect to spire server through node api for workload certificate requests.

+ Edge hub implements spire workload api to retrieve certificates. Cloud spire agent (at edge node) performs workload attestation. On successful attestation, certificate signing request is made to cloud spire server. On response from cloud spire server, cloud spire agent caches the certificate and responds to cloud hub with private key and certificate.

+ Edge spire server initialization can be performed with two options. 
1) (Recommended) Edge spire server can download workload api certificates and connect to edgehub (or metamanager?) to download intermediate CA certificates.
2) Edge spire server can download workload api certificates and connect to cloud spire server directly to download intermediate CA certificates.

+ EdgeD on initialization creates node entry with edge spire server and updates the status to kubernetes api server through hub interface.

<img src="../images/security/SpireEdgev0.1-CreateEdgeNodeSequence.png">

To delete node,

<img src="../images/security/SpireEdgev0.1-DeleteNodeSequence.png">

### Application registration and deletion

On addition of a new user application pod, edged creates entry at edge node spire server, which enables application to retrieve and rotate certificates.

<img src="../images/security/SpireEdgev0.1-CreatePodSequence.png">

For delete pod,

<img src="../images/security/SpireEdgev0.1-DeletePodSequence.png">

## Offline scenarios
When there is intermittent / no connectivity between the edge node and the cloud , edge applications can still rotate certificates based on TTL configured for edge spire server.

## Scalability
In the current design, the solution is scalable for single cluster of edge application deployment under a node. Applications hosted under different edge spire server requires a federated spire server deployment for east-to-west direct communication between applications. Also, a policy management framework is necessary for controlling the authorization between applications.

## Open questions
- How are edge nodes identified from other kubernetes nodes deployed in a cluster (heterogenous deployments) ?
- Is it better to use a different CRD for managing identity entries? Using a CRD requires an extra API call to manage the entries.
- Is it better to consider zero trust environment and enable encryption for all communications inside an edge/cloud node?
