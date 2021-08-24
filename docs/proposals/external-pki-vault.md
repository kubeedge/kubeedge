---
title: External PKI Support
status:
authors:
    - "@subpathdev"
approvers:
creation-date: 2021-09-11
last-updated: 2021-09-11
---

## Table of Contents

- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Use Cases](#use-cases)
- [Changes](#changes)

## Motivation

The usage of an external public key infrastructure (PKI) allows users a comfortable way to manage their certificates.
Additional this usage allows us to delegate the responsibility of the correct usage of the certificates to the
users. A separate feature is that, the usage of valid certificates are possible.
This feature allows us to increase the security of the hole system, by using a certificate revocation list (CRL)
or certificates with a short duration.

### Goals

- [Hashicorp Vault](https://www.vaultproject.io/) can be used to manage certificates
- other PKIs can be added by implementing an interface

### Non-Goals

- adding support for every external PKI
- implementing a full PKI, which can be used by kubeedge

### Use Cases

There are different use cases which cannot be used with the current architecture.
By using a PKI some of the use cases can be fulfilled.

A few of those use cases will be describe in this chapter.

#### Theft

An edge node, which is added by using KubeEdge can be stolen. Users should be able to remove those nodes
easily. This can already be made with the current setup by using `kubectl delete node <name of the stolen edge node>`
The certificates are still on the edge node. So they are able to join the cluster again. This can be stopped by using
a CRL where the certificate of the node can be placed. Another way to handle this, is by using short times when the
certificate is valid. A token, which can also be revoked, can also be used to forbid the newly creation.

#### Multi Tenant System
In a multi tenant system, where customers have a monthly based subscription, a PKI can be used to control the entrance
to the system. They can upload data through the edgecore and the cloudcore to the system, so that the edgecore needs
a valid certificate every month. In these use case a PKI is also be helpful.

## Changes

In this chapter we want to describe which changes have to be made on the current system and the architecture to make use
an external PKI. In the first step we will describe the changes which will be made to integrate the interface. The second step
we will describe, how the external PKI vault will be added.

### Cloudcore

The first step is to develop a seperate module, which provides the same functionality, as the current system does. By developing this system, an interface will be
created which encapsulate all features. In the second step a different implementation of the interface will developed, which is using hashicorp vault as certificate
requests. Different ways of configuring can be made. Currently only speaking directly with the vault with their REST-API and a Token to authenticate. In this case a
central Vault is needed and has to be used by the cloudcore and all edgecores. The functionality to create or renew certificats can also be used by the edgecore.

### Edgecore

In the edgecore the creation of the certificate will also move to a separate module. This module can create or renew certificates by using the cloucore PKI and in a
second step, the creation and the certificate renew will also be delegated to vault. Both variants are able to automatically add an edgecore by using only a single
token.