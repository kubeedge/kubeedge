title: Support for external PKI for authentication of edge nodes

# Motivation

The current implementation of KubeEdge performs tasks of a Certification Authority,
i.e. cryptographic certificates are created and validated. Additionally, only partial
support for certificate rotation on the edge node side and no support for certificate
rotation on the cloud hub side is provided. By integrating
[Hashicorp Vault](https://www.vaultproject.io/) and offloading the heavy lifting
of certificate handling to it, the framework could be more robust, on point and
more easily to maintain.
This proposal describes an approach for the integration of Vault with some moderate
changes to the implementation. A pull request is currently prepared, that implements
these changes and may be used to demonstrate the viability of the integration approach.

# Driving Forces

The following chapters list the driving forces that guide the proposed changes.

## Secure authentication

The main goal is the secure authentication of the edge nodes: Before a successful communication with the cloudhub is allowed, the _identity_ of the edge node has to be established. The authentication has to be performed without user interaction, as edge nodes are commonly expected to work autonomously without user input.
Currently, two forms of authentication are used:

* Client certificate: A predefined X.509 client certificate is passed in the configuration to the edgenode. This certificate is used to authenticate the edgenode to the cloud hub, the communication may only proceed, if the certificate is valid and signed by a certification authority known to the cloudhub. The main disadvantage of this approach is, that currently no automatic mechanisms for revoking or replacing this certificate are available.

* Bearer token that is provided manual via configuration: The edgenode presents a pre-configured JWT bearer token to authenticate itself. Using this token the cloudhub _generates_ a new client certificate that is provided to the edgenode. Further authentication is done using this certificate. The obvious issues with this approach are
    * The JWT token must be configured manually
    * No token refresh mechanism is defined
    * The cloudhub must perform tasks of a certification authority with obvious security impacts

## Moving security functionality to external facilities

Currently, considerable amounts of code within KubeEdge perform authentication and authorization functionality, e.g. the cloudhub may function as a _certification authority_ to create new (client-) X.509 certificates. This requires the storing and handling of sensitive data, e.g. private cryptographic keys. Furthermore, traditionally the correct implementation of security functionality is non-trivial. Therefore it is advisable, to move as much security relevant functionality to systems that are explicitly implemented for this kind of operations. Removing, resp. _deprecating_, security relevant implementations and sensitive data considerably reduces the attack surfaces of the overall system.

## Autonomous Renew

To adequate security it is important, that credentials for authentication are not hardcoded for long term usage, but periodically refreshed. For IoT devices it is especially important, that this is done _autonomously_, i.e. executed without further user interaction. This requires, that

* X.509 client certificates and JWT tokens have a restricted lifetime to ensure regular renewal
* An automatic process exists, that allows to easily renew these credentials.

A challenge for this is, that the identity of the requesting edgenode has to be established, before a fresh client certificate may be provided. Therefore, a _bootstrapping_ process is required.

## Revocation and Blocking of Unauthorized Access

Mobile IoT devices are at risk of getting lost or stolen. In this case it is desirable, to lock the access to the cloudhub of this device as soon as possible. The main building blocks for this are

* Using only _short-lived_ credentials for the device, so that a refresh is required periodically
* Block the facility for refreshing the credentials as soon as possible to minimize the attack surface for possible rogue devices.

Additionally it is advisable to only _lock_ the access for the "lost" device, as a recovered device may be unlocked and reused again.
This

## Backwards Compatibility

Currently, the KubeEdge frameworks provides a considerable amount of functionalities to ensure authentication and authorization. Consequently, a lot of configuration and implementation already exist that rely on this functionality. Therefore it is essential, that the current functionally _remains backward compatibility_ to protect existing investments in setup and operation of KubeEdge. This requires, that all new functionality is _added side-by-side_ with the existing functions. So,

* If needed, new configuration settings must be added without changing the syntax or semantics of already existing settings.
* Suitable default values must be provided for new functionalities that ensure, that the new functionality does not interfere with existing configurations. Optimally, all new functionality should be _disabled_ by default.

## Tamper proof

As the identity of the edgenode is the root of all security aspects for the system, it is essential, that the identity information of the device is stored as secure as possibly.

# Architecture of the Proposed Change

## Motivation

As discussed above, it would be advisable to move security relevant functions and data to a dedicated system, that is designed for such tasks and explicitly hardened. One example of such a system would be [Hashicorp Vault](https://www.vaultproject.io/docs). This system may be used as _security module_ that

* securely stores sensitive data, e.g. private keys
* may be used as a _certification authority_ with all associated functionality:
    * Issuing of new certificates
    * Revocation of certificates
    * Facilities to implement certificate renewals
* Authentication mechanisms
* Secret management

Additionally vault is a cloud friendly application that may be operated highly available within a kubernetes cluster.

## High Level Overview

The following diagram shows the high level overview of the system:

 ![](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/edgefarm/kubeedge/vault/docs/images/proposals/pki-system-components.puml)

 Within the cloud environment an instance of vault is made available.

Vault will perform the following tasks:

* For the edge hub:
    * Authenticate the cloud hub pods for certificate handling
    * Provide X.509 server certificates for the edge node interfaces
    * Provide periodic X.509 certificate renewal for the server certificates
    * Revoke edge hub client certificates
    * Provide revocation lists for edge hub client certificates
* For the cloud hub:
    * Authenticate the edge hub nodes for certificate handling and to establish the identity of the cloud hub node
    * Allow creation of X.509 client certificates to secure the communication with the cloud hub. The certificates must be signed with a ca certificate that allows the cloud hub to validate the client certificate
    * Allow renewal of the client certificate

The security system of Vault allows it to define in fine granularity, which client may invoke which operation. For example, it can be defined that a client may retrieve a newly generated client certificate, however, the X.500 common name and additional attributes, e.g. the TTL, of the certificate are fixed to predefined values.

# Use Cases
## Cloud Hub

These are detailed use cases for the cloud hub

### Authenticate to Vault

As a first step, the cloud hub pods have to authenticate to Vault in order to gain the privileges for all following use cases. This
requires to make a _Vault login token_ available to the application. This requires
some manual effort, as the commonly used  methods used by Vault ([agent injector](https://www.vaultproject.io/docs/platform/k8s/injector#agent-sidecar-injector) resp. [container storage injection](https://www.vaultproject.io/docs/platform/k8s/csi#vault-csi-provider))
cannot be used here (they are intended to inject _single_ secrets, i.e. credentials. They cannot be used to inject _both_ a new
certificate and a private key). So authentication of the cloud core may be implemented by using a _sidecar_ container that

* Uses a serviceaccount to authenticate to vault via the [kubernetes authentication method](https://www.vaultproject.io/docs/auth/kubernetes#kubernetes-auth-method)
* When the identity of the cloudcore pods has been established, the sidecar container may retrieve new certificates
from the Vault server (restricted by Vault policies) and place them into a shared volume, where the cloudcore pod
may access them via simple file access.
* The cloudcore server may use the _GetCertificate_ functionpointer of [TLSConfig](https://pkg.go.dev/crypto/tls#Config)
to present a recent certificate to connecting clients.

The detailed flow can be implemented as shown in the following diagram:

![](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/edgefarm/kubeedge/vault/docs/images/proposals/pki-cloudhub-auth.puml)


#### Implementation notes
Note, that the certificates must already be present when the cloudhub containers starts, as it tries to retrieve the certificates
from the filesystem. A pragmatic approach to ensure this is to define the sidecar redundantly:

* First as an _init container_ that is run before the main containers start. The init container may then retrieve the initial certificates
* As standard sidecar container that periodically refreshes the certificates

This will lead to some redundancy in the deployment descriptor, but ensures the presence of valid certificates during the lifetime
of the cloudhub container.

For the implementation of the cloudhub container the following changes have to be performed:

* Currently the implementation requires the presence of the CA certificate as well as the _private key_. As the private key for the CA
certificate now remains within Vault, the validation on startup have to be relaxed for this
* When using Vault, it is not necessary any more, to create and maintain a separate _kubernetes secrets_ containing certificates and
secret keys anymore.
* The servers (Websocket, QUIC) must use _dynamic_ secret resolution using the TLS configuration (supported by the Go API). This will
present the current certificate to connecting client, therefore allowing transparent certificate renewal.
* Additionally, a certificate renewal by _dropping_ the current connections and forcing the clients to reconnect. Active TLS connections
do not allow for a intermittent certificate renewal. However, this would be disruptive for all current



### Validate a Client Certificate

When a client connects to the cloud hub http server, the provided client certificate must be validated:

* Validity period is not stale
* Serial number of the offered client certificate has not been revoked
* The passed certificate is signed by a valid certificate chain

![](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/edgefarm/kubeedge/vault/docs/images/proposals/pki-cloudhub-clientvalidation.puml)

The validation of the client interface must be done programmatically on establishing the connection and is done by the client libraries. Vault does not provide functionality for _validating a certificate chain_. However, the TLS Server functionality provided by Go already implements client certificate validation.

## Automatic Renewal

The server certificate should be periodically renewed. An automatic job may retrieve a new certificate and replace the one currently used. As the certificate is signed by a valid certification authority, the certificate change will be transparent for the connecting edge nodes.
As mentioned above, the side car container periodically retrieves a new certificate and provides it via a shared volume.


## Edge Hub

These are detailed use cases for the edge hub

### Authenticate to Vault

The initial step for the edge hub nodes is to establish their identity with Vault. For this, the edge node requires an _initial Vault token_ that allows it to authenticate itself to Vault. When authenticated, the edgenode can retrieve a x.509 certificate to secure the connection to the cloudhub.
The authentication is straight forward:
 ![](http://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/edgefarm/kubeedge/vault/docs/images/proposals/pki-edgehub-auth.puml)
 The token is checked by Vault for validity, when required the requesting client is rejected and will not be able to communicate the cloudedge server.

### Renew the token

For this approach, the Vault token should be long lived. However, it is also possible to use shorter lived tokens and _renew_ this token periodically. As the token is stored in the filesystem, the token renewal may be implemented by simply replacing this file (atomically).
### Generate a Client Certificate

Using the bearer token the edge hub may request a new certificate. This requires, that

* A appropriate _Vault role_ has been defined, that defines the parameters of the new X.509 certificate, e.g. common name and validity period
* A _Vault policy_ has been defined, that only allows requesting a certificate for the correct name of the requesting edge node

It is recommended to keep the validity period of the generated certificate as short as possible, e.g. a single day.

#### Implementation Note
Unfortunately, due to the architecture of the edgenode daemon, it is not possible to use the same approach as for the cloudhub server, i.e. to use a _sidecar_ container that implements the authentication and certificate handling. It it necessary, to implement the same functionality within the edge hub itself (however, a suitable library can be used to utilize the same implementation). Consequently, the _configuration_ of the edge hub has to be extended as well to configure Vault related parameters.

The following settings will have to be added:


 Parameter | Default | Description
 --- | --- | ---
 Enable | false | Enable the Vault specific functionality
Tokenfile | n/a | The file containing the Vault identity token
CommonName | n/a | The common name for the certificates requested. Note that the Vault server may be configured to restrict the valid common names
TTL | n/a | The validity period for the requested certificates. Note that the Vault server may be configured to impose additional restrictions on the maximum lifetime.


### Renew a Client Certificate

As the client certificate is short lived, it requires periodic renewal. This is identical the generation of a new certificate.

## Device enrollment

The enrollment of a new device is straight forward. Basically, as the actual functionality is part of the edge hub server itself, only a suitable configuration (see above) and a Vault token has to provisioned. The generation of the token is out of scope for this proposal.