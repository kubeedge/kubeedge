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

We expect our proposed device management and cloud-edge communication standards will be adopted by mainstream IoT edge cloud providers and KubeEdge becomes the best possible platform for IoT and Edge computing.

### 2.2 Out of Scope

We will not try to go deeper into general IoT and Edge computing discussions as there are excellent papers that already cover these topics in details we are a open source software community.

## 3. Roles and Organization Management

- Proposing and making decisions *MAY* be done without the use of KEPS so long as the decision is documented in a linkable medium. We prefer to see written decisions and reasoning on the [kubeedge@](https://groups.google.com/forum/#!forum/kubeedge) mailing list or as issues filed against [kubeedge/kubeedge](https://github.com/kubeedge/kubeedge). We encourage the use of faster mediums such as slack of video conferences to come to consensus.