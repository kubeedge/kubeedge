# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3018/badge)](https://bestpractices.coreinfrastructure.org/projects/3018)

<img src="./docs/images/kubeedge-logo-only.png">

English | [简体中文](./README_zh.md)

KubeEdge is built upon Kubernetes and extends native containerized application orchestration and device management to hosts at the Edge.
It consists of cloud part and edge part, provides core infrastructure support for networking, application deployment and metadata synchronization
between cloud and edge. It also supports **MQTT** which enables edge devices to access through edge nodes.

With KubeEdge it is easy to get and deploy existing complicated machine learning, image recognition, event processing and other high level applications to the Edge.
With business logic running at the Edge, much larger volumes of data can be secured & processed locally where the data is produced.
With data processed at the Edge, the responsiveness is increased dramatically and data privacy is protected.

KubeEdge is an incubation-level hosted project by the [Cloud Native Computing Foundation](https://cncf.io) (CNCF). KubeEdge incubation [announcement](https://www.cncf.io/blog/2020/09/16/toc-approves-kubeedge-as-incubating-project/) by CNCF.

**Note**:

The versions before *1.8* have not been supported, please try upgrade.

## Advantages

- **Kubernetes-native support**: Managing edge applications and edge devices in the cloud with fully compatible Kubernetes APIs.
- **Cloud-Edge Reliable Collaboration**: Ensure reliable messages delivery without loss over unstable cloud-edge network.
- **Edge Autonomy**: Ensure edge nodes run autonomously and the applications in edge run normally, when the cloud-edge network is unstable or edge is offline and restarted.
- **Edge Devices Management**: Managing edge devices through Kubernetes native APIs implemented by CRD.
- **Extremely Lightweight Edge Agent**: Extremely lightweight Edge Agent(EdgeCore) to run on resource constrained edge.


## How It Works

KubeEdge consists of cloud part and edge part.

### Architecture

<div  align="center">
<img src="./docs/images/kubeedge_arch.png" width = "85%" align="center">
</div>

### In the Cloud
- [CloudHub](https://kubeedge.io/en/docs/architecture/cloud/cloudhub): a web socket server responsible for watching changes at the cloud side, caching and sending messages to EdgeHub.
- [EdgeController](https://kubeedge.io/en/docs/architecture/cloud/edge_controller): an extended kubernetes controller which manages edge nodes and pods metadata so that the data can be targeted to a specific edge node.
- [DeviceController](https://kubeedge.io/en/docs/architecture/cloud/device_controller): an extended kubernetes controller which manages devices so that the device metadata/status data can be synced between edge and cloud.


### On the Edge
- [EdgeHub](https://kubeedge.io/en/docs/architecture/edge/edgehub): a web socket client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge Architecture). This includes syncing cloud-side resource updates to the edge, and reporting edge-side host and device status changes to the cloud.
- [Edged](https://kubeedge.io/en/docs/architecture/edge/edged): an agent that runs on edge nodes and manages containerized applications.
- [EventBus](https://kubeedge.io/en/docs/architecture/edge/eventbus): a MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- [ServiceBus](https://kubeedge.io/en/docs/architecture/edge/servicebus): a HTTP client to interact with HTTP servers (REST), offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge.
- [DeviceTwin](https://kubeedge.io/en/docs/architecture/edge/devicetwin): responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- [MetaManager](https://kubeedge.io/en/docs/architecture/edge/metamanager): the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite).

## Kubernetes compatibility

|                        | Kubernetes 1.16 | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20 | Kubernetes 1.21 | Kubernetes 1.22 |
|------------------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|
| KubeEdge 1.10          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge 1.11          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge 1.12          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge HEAD (master) | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |

Key:
* `✓` KubeEdge and the Kubernetes version are exactly compatible.
* `+` KubeEdge has features or API objects that may not be present in the Kubernetes version.
* `-` The Kubernetes version has features or API objects that KubeEdge can't use.

## Guides

Get start with this [doc](https://kubeedge.io/en/docs).

See our documentation on [kubeedge.io](https://kubeedge.io) for more details.

To learn deeply about KubeEdge, try some examples on [examples](https://github.com/kubeedge/examples).

## Roadmap

* [2021 Roadmap](./docs/roadmap.md#roadmap)

## Meeting

Regular Community Meeting:
- Europe Time: **Wednesdays at 16:30-17:30 Beijing Time** (biweekly, starting from Feb. 19th 2020).
([Convert to your timezone.](https://www.thetimezoneconverter.com/?t=16%3A30&tz=GMT%2B8&))
- Pacific Time: **Wednesdays at 10:00-11:00 Beijing Time** (biweekly, starting from Feb. 26th 2020).
([Convert to your timezone.](https://www.thetimezoneconverter.com/?t=10%3A00&tz=GMT%2B8&))

Resources:
- [Meeting notes and agenda](https://docs.google.com/document/d/1Sr5QS_Z04uPfRbA7PrXr3aPwCRpx7EtsyHq7mp6CnHs/edit)
- [Meeting recordings](https://www.youtube.com/playlist?list=PLQtlO1kVWGXkRGkjSrLGEPJODoPb8s5FM)
- [Meeting link](https://zoom.us/j/4167237304)
- [Meeting Calendar](https://calendar.google.com/calendar/embed?src=8rjk8o516vfte21qibvlae3lj4%40group.calendar.google.com) | [Subscribe](https://calendar.google.com/calendar?cid=OHJqazhvNTE2dmZ0ZTIxcWlidmxhZTNsajRAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ)

## Contact

If you need support, start with the [troubleshooting guide](https://kubeedge.io/en/docs/developer/troubleshooting), and work your way through the process that we've outlined.

If you have questions, feel free to reach out to us in the following ways:

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)
- [slack](https://join.slack.com/t/kubeedge/shared_invite/enQtNjc0MTg2NTg2MTk0LWJmOTBmOGRkZWNhMTVkNGU1ZjkwNDY4MTY4YTAwNDAyMjRkMjdlMjIzYmMxODY1NGZjYzc4MWM5YmIxZjU1ZDI)
- [twitter](https://twitter.com/kubeedge)

## Contributing

If you're interested in being a contributor and want to get involved in
developing the KubeEdge code, please see [CONTRIBUTING](./CONTRIBUTING.md) for
details on submitting patches and the contribution workflow.

## Security

### Security Audit

A third party security audit of KubeEdge has been completed in July 2022. Additionally, the KubeEdge community completed an overall system security analysis of KubeEdge. The detailed reports are as follows.

- [Security audit](https://github.com/kubeedge/community/blob/master/sig-security/sig-security-audit/KubeEdge-security-audit-2022.pdf)

- [Threat model and security protection analysis paper](https://github.com/kubeedge/community/blob/master/sig-security/sig-security-audit/KubeEdge-threat-model-and-security-protection-analysis.md)

### Reporting security vulnerabilities

We encourage security researchers, industry organizations and users to proactively report suspected vulnerabilities to our security team (`cncf-kubeedge-security@lists.cncf.io`), the team will help diagnose the severity of the issue and determine how to address the issue as soon as possible.

For further details please see [Security Policy](https://github.com/kubeedge/community/blob/master/team-security/SECURITY.md) for our security process and how to report vulnerabilities.

## License

KubeEdge is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
