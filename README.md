# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)


<img src="./docs/images/KubeEdge_logo.png">

KubeEdge is an open source system extending native containerized application orchestration and device management to hosts at the Edge. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. It also supports **MQTT** and allows developers to author custom logic and enable resource constrained device communication at the Edge. KubeEdge consists of a cloud part and an edge part.

## Advantages

#### Edge Computing

With business logic running at the Edge, much larger volumes of data can be secured & processed locally where the data is produced. Edge nodes can run autonomously which effectively reduces the network bandwidth requirements and consumptions between Edge and Cloud. With data processed at the Edge, the responsiveness is increased dramatically and data privacy is protected.

#### Simplified development

Developers can write regular http or mqtt based applications, containerize them, and run them anywhere - either at the Edge or in the Cloud - whichever is more appropriate.

#### Kubernetes-native support

With KubeEdge, users can orchestrate apps, manage devices and monitor app and device status on Edge nodes just like a traditional Kubernetes cluster in the Cloud. Locations of edge nodes are transparent to customers.

#### Abundant applications

It is easy to get and deploy existing complicated machine learning, image recognition, event processing and other high level applications to the Edge.

## Introduction

KubeEdge is composed of the following components:

- [Edged](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/edge/edged.md): an agent that runs on edge nodes and manages containerized applications.
- [EdgeHub](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/edge/edgehub.md): a web socket client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge Architecture). This includes syncing cloud-side resource updates to the edge, and reporting edge-side host and device status changes to the cloud.
- [CloudHub](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/cloud/cloudhub.md): a web socket server responsible for watching changes at the cloud side, caching and sending messages to EdgeHub.
- [EdgeController](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/cloud/controller.md): an extended kubernetes controller which manages edge nodes and pods metadata so that the data can be targeted to a specific edge node.
- [EventBus](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/edge/eventbus.md): a MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- ServiceBus: a HTTP client to interact with HTTP servers (REST), offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge.
- [DeviceTwin](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/edge/devicetwin.md): responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- [MetaManager](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/edge/metamanager.md): the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite). 

### Architecture

<img src="./docs/images/kubeedge_arch.png">

## Usage

* [One click KubeEdge Installer to install both Cloud and Edge nodes](./docs/setup/installer_setup.md)
* [Run KubeEdge from release package](./docs/getting-started/release_package.md)
* [Run KubeEdge from source](./docs/getting-started/usage.md)
* [Deploy Application](./docs/getting-started/usage.md#deploy-application)
* [Run Tests](./docs/getting-started/usage.md#run-tests)

## Roadmap

* [Release 1.0 onwards](./docs/getting-started/roadmap.md#release-1.0-onwards)
* [Release 0.3](./docs/getting-started/roadmap.md#release-0.3) - Plan to Release on 31 May 2019

## Meeting

Regular Community Meeting: Wednesday at 11:00 Beijing Time (biweekly).

- [Meeting notes and agenda](https://docs.google.com/document/d/1Sr5QS_Z04uPfRbA7PrXr3aPwCRpx7EtsyHq7mp6CnHs/edit)
- [Meeting recordings](https://www.youtube.com/playlist?list=PLQtlO1kVWGXkRGkjSrLGEPJODoPb8s5FM)
- [Meeting link](https://zoom.us/j/4167237304)

## Documentation

The detailed documentation for KubeEdge and its modules can be found at [https://docs.kubeedge.io](https://docs.kubeedge.io). 
Some sample applications and demos to illustrate possible use cases of KubeEdge platform can be found at this [examples](https://github.com/kubeedge/examples) repository.

## Contact

<!--
We don't have a troubleshooting guide yet.  When we do, uncomment the following and add the link.
If you need support, start with the [troubleshooting guide], and work your way through the process that we've outlined.

--> 
If you have questions, feel free to reach out to us in the following ways:

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)
- [slack](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc)

## Contributing

If you're interested in being a contributor and want to get involved in
developing the KubeEdge code, please see [CONTRIBUTING](CONTRIBUTING.md) for
details on submitting patches and the contribution workflow.

## License

KubeEdge is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
