# KubeEdge Community Charter

The mission of the KubeEdge community is to discuss, design and document taking advantage of Kubernetes primitives for developing and deploying IoT and Edge specific applications. Meanwhile, address the challenges of deploying Kubernetes in the Edge.

## 1. Goals

Cloud native computing paradigm is the main focus of many development teams these days and there’s a good reason for that. However, there’s another class of applications that on the first look don’t belong to this world. IoT and Edge applications have a lot of distributed components that don’t usually sit together within the same data center infrastructure. On the other hand developers of those applications would benefit greatly of the concepts, infrastructure and tools that are being developed in the cloud-native universe.

The goal of this community is to try to bridge cloud-native and edge computing closer together and lay the foundations for the future work.

## 2. Scope

To achieve this goal, the group will need to deal with the following items:

* Define basic concepts, use cases and architectures used today in IoT/Edge.

- Provide reference architectures for making Kubernetes the unified control plane for IoT/Edge.
- Create and maintain conformance tests tailored towards performance and reliability requirements of the most popular IoT/Edge use cases.
- Describe challenges that exist today for deploying some of the workload and use cases.
- Extend network infrastructure to better suite IoT/Edge use cases over bandwidth constrained and unreliable WAN interconnects.
- Improve connectivity and data ingestion options to better support various field protocols.
- Improve and extend existing CLI tools to manage Kubernetes clusters running in remote edge locations.
- provides necessary technologies to govern and secure IoT devices.

### 2.1 Success Criteria

First of all, KubeEdge is designed to address IoT/Edge workload scenarios. We'd like to enable customers manage node & device resources from cloud. With single pane of glass, IT administrators or users can fulfill the needs the same way as resources in data center.

Secondly, KubeEdge is an open architecture based on Kubernetes. We'd like folks from industry, academy, community etc. to contribute and make innovation.

Thirdly, KubeEdge is currently considered as one of the reference architecture. It is our goal to make the cloud/edge communication channel and device management APIs be the standard and acquire more companies to adopt.

### 2.2 Out of Scope

We will not try to go deeper into general IoT and Edge computing discussions as there are excellent papers that already cover these topics in details and we are a open source software community.

## 3. Roles and Organization Management

### Roles

#### Maintainers

  - Run operations and processes governing the SIG
  - Number: 2-3
  - Membership tracked in [MAINTAINERS](https://github.com/kubeedge/kubeedge/blob/master/MAINTAINERS)

#### Member

- *SHOULD* maintain health of the community
- *SHOULD* show sustained contributions to at least one project or to the community
- *SHOULD* hold some documented role or responsibility in the community and / or at least one project 
- *MAY* build new functionality for projects
- *MAY* participate in decision making for the projects they hold roles in

#### Security Contact

- *MUST* be a contact point for the Product Security Committee to reach out to for triaging and handling of incoming issues
- *MUST* accept the [Embargo Policy](https://git.k8s.io/security/private-distributors-list.md#embargo-policy)
- Defined in `SECURITY_CONTACTS` files, this is only relevant to the root file in the repository. Template [SECURITY_CONTACTS](https://github.com/kubernetes/kubernetes-template-project/blob/master/SECURITY_CONTACTS)

### Organizational Management

- Community members meets bi-weekly on zoom with agenda in meeting notes
  - *SHOULD* be facilitated by chairs unless delegated to specific Members
- Project overview and deep-dive sessions organized for KubeCon/CloudNativeCon
  - *SHOULD* be organized by chairs unless delegated to specific Members
- Contributing instructions defined in the CONTRIBUTING.md

#### Project Management
In addition, community have the following responsibilities to PM:

* identify community annual roadmap
* identify all community features in the current release
* actively track / maintain SIG features within docs/proposals

#### Technical processes

Projects of the community MUST use the following processes unless explicitly following alternatives they have defined.

* Proposing and making decisions

   - Proposals sent as KEP PRs and published to googlegroup as announcement
   - Follow KEP decision making process

* Test health

- Canonical health of code published to
- Consistently broken tests automatically send an alert to
- SIG members are responsible for responding to broken tests alert. PRs that break tests should be rolled back if not fixed within 24 hours (business hours).
- Test dashboard checked and reviewed at start of each SIG meeting. Owners assigned for any broken tests. and followed up during the next SIG meeting.

Proposing and making decisions *MAY* be done without the use of KEPS so long as the decision is documented in a linkable medium. We prefer to see written decisions and reasoning on the [kubeedge@](https://groups.google.com/forum/#!forum/kubeedge) mailing list or as issues filed against [kubeedge/kubeedge](https://github.com/kubeedge/kubeedge). We encourage the use of faster mediums such as slack of video conferences to come to consensus.