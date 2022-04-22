# Self-assessment
This document details the design goals and security implications of KubeEdge to aid in the security assessment by CNCF TAG-Security.

# Self-assessment outline

## Table of contents

* [Metadata](#metadata)
  * [Security links](#security-links)
* [Overview](#overview)
  * [Background](#background)
  * [Goals](#goals)
  * [Non-goals](#non-goals)
* [Self-assessment use](#self-assessment-use)
* [Security functions and features](#security-functions-and-features)
* [Project compliance](#project-compliance)
* [Secure development practices](#secure-development-practices)
* [Security issue resolution](#security-issue-resolution)
* [Appendix](#appendix)

## Metadata

|                   |                                                              |
| ----------------- | ------------------------------------------------------------ |
| Software          | https://github.com/kubeedge/kubeedge                         |
| Website           | https://kubeedge.io                                          |
| Security Provider | No                                                           |
| Languages         | Go                                                           |
| SBOM              | Check [go.mod](https://github.com/kubeedge/kubeedge/blob/master/go.mod) for libraries, packages, versions used by the project |

### Security links

| Doc                          | url                                                          |
| ---------------------------- | ------------------------------------------------------------ |
| Security file                | [SECURITY.md](https://github.com/kubeedge/kubeedge/blob/master/SECURITY.md) |
| Default and optional configs | [cloudcore](https://github.com/kubeedge/kubeedge/blob/master/pkg/apis/componentconfig/cloudcore/v1alpha1/default.go)<br />[edgecore](https://github.com/kubeedge/kubeedge/blob/master/pkg/apis/componentconfig/edgecore/v1alpha1/default.go) |

## Overview

KubeEdge is an open source system for extending native containerized application orchestration capabilities to hosts at Edge. It's built upon Kubernetes and provides fundamental infrastructure support for network, application deployment and metadata synchronization between cloud and edge.

Since joining CNCF, KubeEdge has attracted more than [954 Contributors](https://kubeedge.devstats.cncf.io/d/18/overall-project-statistics-table?orgId=1) from 75 plus different Organizations with 3700+ Commits; got 5100+ Stars on Github and 1570+ Forks. It has been adopted by [Raisecom](https://github.com/kubeedge/kubeedge/blob/master/ADOPTERS.md#raisecom-technology-coltd), [WoCloud](https://cucc.wocloud.cn/), [Xinghai IoT](https://github.com/kubeedge/kubeedge/blob/master/ADOPTERS.md#xinghai-iot), [KubeSphere](https://kubesphere.io/), [HUAWEI CLOUD](https://huaweicloud.com/) etc.

### Background

With the rapid development of cloud, 5G and AI technologies, enterprises are increasingly demanding intelligent upgrading, and the application scenarios of edge computing are becoming more and more extensive. It is challenging to realize unified management and control of edge computing resources and cloud-edge synergy, such as how to run intelligent applications and algorithms on edge devices with limited resources(e.g. cameras and drones), how to solve the problems caused by the access of massive heterogeneous edge devices in intelligent transportation and intelligent power scenarios, and how to ensure high reliability of services in off-line scenarios.

KubeEdge provides solutions for cloud-edge synergy and has been widely adopted in industries including transportation, energy, Internet, CDN, manufacturing, smart campus etc. KubeEdge provides:

- Seamless Cloud-Edge Communication for both metadata and data.
- Edge Autonomy: Autonomous operation of Edge even during disconnection from cloud.
- Low Resource Readiness: KubeEdge can work in constrained resource situations (low memory, low bandwidth, low compute).
- Simplified Device Communication: Easy communication between application and devices for IOT and IIOT.

### Goal

The main goals of KubeEdge are as follows:

- Building an open edge computing platform with cloud native technologies.
- Helping users extending their business architecture, applications, services, etc. from cloud to edge in same experience.
- Implementing extensible architecture based on Kubernetes.
- Integration with CNCF projects, including (but not limited to) containerd, cri-o, Prometheus, Envoy, etc.
- Seamless development, deployment and run complex workloads at edge with optimized resources.

### Non-goals

The scenarios solved by KubeEdge do not include the problems of getting through the underlying physical network of edge nodes and ensuring the reliability communication of cloud side network infrastructure. However, for the scenarios where edge network is off-line, KubeEdge provides the autonomy of edge off-line nodes.

### Project & Design

The following diagram shows the logical architecture for KubeEdge: 

<img src="../images/kubeedge_overview.png">

KubeEdge consists of below major components:

- In the cloud (CloudCore)
  - CloudHub: a websocket server responsible for watching changes at the cloud side, caching and sending messages to EdgeHub.
  - EdgeController: an extended kubernetes controller which manages edge nodes and pods metadata so that the data can be targeted to a specific edge node.
  - DeviceController: an extended kubernetes controller which manages devices so that the device metadata/status data can be synced between edge and cloud.
  - SyncController: periodically check data that persists on edge and cloud, trigger reconcile if necessary.
  - DynamicController: an extended kubernetes controller which is based on Kubernetes dynamic client, allows client on the edge node list/watch any common Kubernetes resource and custom resources.
- On the edge (EdgeCore)
  - EdgeHub: a websocket client responsible for interacting with Cloud Service for edge computing (EdgeHub and CloudHub are symmetric components for edge cloud communication). This includes syncing cloud-side resource updates to the edge, and reporting edge-side host and device status changes to the cloud.
- MetaManager: the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite).
  - Edged: an agent that runs on edge nodes and manages containerized applications.
- DeviceTwin: responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
  - EventBus: a MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- ServiceBus: a HTTP client to interact with HTTP servers (REST), offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge.
  

## Self-assessment use

This self-assessment is created by the KubeEdge team to perform an internal analysis of the project's security.  It is not intended to provide a security audit of KubeEdge, or function as an independent assessment or attestation of KubeEdge's security health.

This document serves to provide KubeEdge users with an initial understanding of KubeEdge's security, where to find existing security documentation, KubeEdge plans for security, and general overview of KubeEdge security practices, both for development of KubeEdge as well as security of KubeEdge.

This document provides the CNCF TAG-Security with an initial understanding of KubeEdge to assist in a joint-review, necessary for projects under incubation.  Taken together, this document and the joint-review serve as a cornerstone for if and when KubeEdge seeks graduation and is preparing for a security audit.

## Security functions and features

* The cloud component CloudHub exposes ports to provide external services, accept connections from edge nodes, and be responsible for cloud-side communication. WebSocket or Quick protocol is used for data transmission, and TLS certificate is used for data encryption. Communications between the CloudHub server and the edge node should be authenticated and encrypted to ensure that attackers who may be in a network position to view or modify this traffic cannot do so. To achieve this access the CloudHub server must be using certificates from a trusted certificate authority so that they can validate their mutual identities.

* The MetaServer module provide list-watch capability externally on the edge side. It’s important to ensure that the RBAC rights of a service account attached in pods are restrictedly configured.


## Project compliance

Not applicable

## Secure development practices

KubeEdge has achieved the passing level criteria for [CII Best Practices](https://bestpractices.coreinfrastructure.org/en/projects/3018).

- Development Pipeline

All code is maintained in [Git](https://github.com/kubeedge/kubeedge) and changes must be reviewed by maintainers and must pass all unit and e2e tests. Code changes are submitted via Pull Requests (PRs) and must be signed. Commits to the `main` branch directly are not allowed.

- Communication Channels
  - Internal. How do team members communicate with each other?
    Team members communicate with each other frequently through [Slack Channel](https://join.slack.com/t/kubeedge/shared_invite/enQtNjc0MTg2NTg2MTk0LWJmOTBmOGRkZWNhMTVkNGU1ZjkwNDY4MTY4YTAwNDAyMjRkMjdlMjIzYmMxODY1NGZjYzc4MWM5YmIxZjU1ZDI), [KubeEdge sync meeting](https://zoom.us/my/kubeedge), and team members will new a [issue](https://github.com/kubeedge/kubeedge/issues) to make a deep discussion if necessary.
  
  - Inbound. How do users or prospective users communicate with the team?
    Users or prospective users usually communicate with the team through [Slack Channel](https://kubeedge.slack.com/archives/CDXVBS085), you can new a [issue](https://github.com/kubeedge/kubeedge/issues) to get further help from the team, and [KubeEdge mailing list](https://groups.google.com/forum/#!forum/kubeedge) is also available. Besides, we have regular [community meeting](https://zoom.us/my/kubeedge) (includes SIG meetings) alternative between Europe friendly time and Pacific friendly time, all these meetings are publicly accessible and meeting records are uploaded to YouTube.

    Regular Community Meeting:
    - Europe Time: **Wednesdays at 16:30-17:30 Beijing Time** (biweekly, starting from Feb. 19th 2020). ([Convert to your timezone.](https://www.thetimezoneconverter.com/?t=16%3A30&tz=GMT%2B8&))
    - Pacific Time: **Wednesdays at 10:00-11:00 Beijing Time** (biweekly, starting from Feb. 26th 2020). ([Convert to your timezone.](https://www.thetimezoneconverter.com/?t=10%3A00&tz=GMT%2B8&))
  
- Outbound. How do you communicate with your users? (e.g. flibble-announce@ mailing list)
    KubeEdge communicate with users through [Slack Channel](https://kubeedge.slack.com/archives/CUABZBD55), [issue](https://github.com/kubeedge/kubeedge/issues), [KubeEdge sync meeting](https://zoom.us/my/kubeedge), [KubeEdge mailing list](https://groups.google.com/forum/#!forum/kubeedge), as for security issues, we provide the communication channels as follows.
  
  - Security email group
    You can email [kubeedge-security](mailto:cncf-kubeedge-security@lists.cncf.io) to report a vulnerability, and the KubeEdge security team will disclose to the distributors through [distributors announce list](mailto:cncf-kubeedge-distrib-announce@lists.cncf.io), see more details [here](https://github.com/kubeedge/kubeedge/security/policy).

- Ecosystem

  KubeEdge Helps users extending their business architecture, applications, services, etc. from cloud to edge in same experience, implements extensible architecture based on Kubernetes and integrates with CNCF projects, including (but not limited to) containerd, cri-o, Prometheus, Envoy, etc. 

  KubeEdge also integrates project kubesphere to align with the cloud native ecosystem. kubesphere is a distributed operating system for cloud-native application management, using Kubernetes as its kernel. It provides a plug-and-play architecture, allowing third-party applications to be seamlessly integrated into its ecosystem. More KubeEdge adopters are listed [here](https://github.com/kubeedge/kubeedge/blob/master/ADOPTERS.md).

## Security issue resolution

### Responsible Disclosures Process

KubeEdge project vulnerability handling related processes are recorded in [Security Policy](https://github.com/kubeedge/kubeedge/security/policy), Related security vulnerabilities can be reported and communicated via email `cncf-kubeedge-security@lists.cncf.io`.

### Incident Response

See the [KubeEdge releases page](https://github.com/kubeedge/kubeedge/releases) for information on supported versions of KubeEdge. Once the fix is confirmed, the Security Team will patch the vulnerability in the next patch or minor release, and backport a patch release into the latest three minor releases.

The release of low to medium severity bug fixes will include the fix details in the patch release notes. Any public announcements sent for these fixes will be linked to the release notes.

## Appendix

### Known Issues Over Time

No known vulnerabilities have been reported.

### [CII Best Practices](https://www.coreinfrastructure.org/programs/best-practices-program/)

KubeEdge has achieved an Open Source Security Foundation (OpenSSF) best practices badge at `passing` level, see more details at [KubeEdge's openssf best practices](https://bestpractices.coreinfrastructure.org/en/projects/3018).

### Case Studies

KubeEdge has been widely adopted in industries including transportation, energy, Internet, CDN, manufacturing, smart campus etc. 

* Xinghai IoT is an IoT company that provides comprehensive smart building solutions by leveraging a construction IoT platform, intelligent hardware, and AI. Xinghai IoT built a smart campus with cloud-edge-device synergy based on KubeEdge and its own Xinghai IoT cloud platform, greatly improving the efficiency of campus management. With AI assistance, nearly 30% of the repetitive work is automated. In the future, Xinghai IoT will continue to collaborate with KubeEdge to launch KubeEdge-based smart campus solutions.
* In China’s highway electronic toll collection (ETC) system, KubeEdge helps manage nearly 100,000 edge nodes and more than 500,000 edge applications in 29 of China’s 34 provinces, cities and autonomous regions. With these applications, the system processes more than 300 million data records daily and has improved traffic efficiency at the toll stations by 10 times.
* China Mobile On-line Marketing Service Center, is a secondary organ of the China Mobile Communications Group, holds the world’s largest call center with 44,000 agents, 900 million users, and 53 customer service centers, builds a cloud-edge synergy architecture consisting of two centers and multiple edges based on KubeEdge. See more details [here](https://www.cncf.io/blog/2021/08/16/china-mobile-kubeedge-based-customer-service-platform-featuring-edge-cloud-synergy).
* e-Cloud of China Telecom, uses KubeEdge to manage CDN edge nodes, automatically deploy and upgrade CDN edge services, and implement edge service disaster recovery (DR) when it migrates its CDN services to the cloud. See more details [here](https://www.cncf.io/blog/2022/03/18/e-cloud-large-scale-cdn-using-kubeedge).

### Related Projects / Vendors

Another edge computing that relative to Kubernetes is OpenYurt, it uses a centralized Kubernetes control plane residing in the cloud site to manage multiple edge nodes residing in the edge sites. OpenYurt manage edge nodes through native Kubelet component, each edge node has moderate compute resources available in order to run edge applications plus the required OpenYurt components. It provides a edge node daemon YurtHub that serves as a proxy for the outbound traffic from typical Kubernetes node daemons such as Kubelet, Kubeproxy, CNI plugins, etc.

KubeEdge is built upon Kubernetes and extends native containerized application orchestration and device management to hosts at the Edge. It consists of cloud part and edge part, provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. In terms of edge components, KubeEdge performs lightweight tailoring for edge resource-constrained scenarios, it requires less node resources and has a wider range of use. KubeEdge also includes a lightweight, edge-centric to access edge elements, provides IoT device access capability. Under the umbrella of KubeEdge project, Sedna realizes the cross-edge cloud collaborative training and collaborative reasoning capabilities of AI, supporting the mainstream AI framework in the industry; EdgeMesh provides simple service discovery and traffic proxy functions for applications, thereby shielding the complex network structure in edge scenarios.