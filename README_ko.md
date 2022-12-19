
# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3018/badge)](https://bestpractices.coreinfrastructure.org/projects/3018)

<img src="./docs/images/kubeedge-logo-only.png">

English | [简体中文](./README_zh.md)| [한국어](./README_ko.md)

KubeEdge는 쿠버네티스 기반으로 되어 있습니다. 네이티브한 컨테이너화된 어플리케이션 오케스트레이션 기능을 엣지에 있는 확장합니다.
기본적으로 클라우드 부분과 엣지 부분으로 구성되어있고, 클라우드와 엣지간 네트워킹, 어플리케이션 배포 및 메타데이터 동기화 기능들을 지원하기 위해 핵심적인 인프라를 제공합니다.
또한, MQTT를 제공해서 엣지 장치들이 엣지 노드에게 접근할 수 있게 합니다.

KubeEdge는 기존의 복잡한 머신러닝, 이미지 인식, 이벤트 처리 및 기타 높은 수준의 어플리케이션을 Edge에 쉽게 가져와 배포할 수 있게 합니다.
엣지에서 실행되는 비즈니스 로직을 통해 훨씬 더 많은 양의 데이터를 보호하고 데이터가 생성되는 로컬에서 처리할 수 있습니다.
데이터를 엣지에서 처리함으로써, 응답속도가 획기적으로 향상되고 동시에 데이터 프라이버시가 보호됩니다.

KubeEdge는 다음 기관에서 호스팅하는 인큐베이션 수준의 프로젝트입니다. [Cloud Native Computing Foundation](https://cncf.io) (CNCF).
KubeEdge 인큐베이션 정보 [announcement](https://www.cncf.io/blog/2020/09/16/toc-approves-kubeedge-as-incubating-project/) by CNCF.

**Note**:

 *1.8* 버전은 더이상 지원되지 않습니다. 업그레이드 해주세요. 

## 강점

- **Kubernetes 네이티브 지원**: 클라우드의 엣지 어플리케이션 및 엣지 장치 관리에 대한 쿠버네티스 API들을 완벽하게 호환 합니다.
- **Cloud-Edge 신뢰가능한 연동**: 불안정한 클라우드-엣지 네트워크 상황에서도 손실없는 메시지 전달을 보장합니다. 
- **Edge의 자율성**: 클라우드-엣지 네트워크가 불안정하거나 엣지가 오프라인 상태에서 다시 시작될 때, 엣지 노드가 어플리케이션을 엣지 위에서 정상적으로 실행되는지 자율적으로 확인하도록 동작합니다. 
- **Edge장치 관리**: CRD로 구현된 쿠버네티스 API를 통해서 엣지 장치들을 관리합니다.
- **초경량 엣지 에이전트**: 성능이 한정적인 엣지위에서 실행하기 위한 초경량 엣지 에이전트를 제공합니다.


## 동작방식

KubeEdge는 클라우드 파트와 엣지 파트로 구성되어 있습니다.

### 아키텍쳐

<div  align="center">
<img src="./docs/images/kubeedge_arch.png" width = "85%" align="center">
</div>

### 클라우드 파트
- [CloudHub](https://kubeedge.io/en/docs/architecture/cloud/cloudhub): 클라우드 사이드에서, 변경사항 감지하여 메시지를 캐싱하고 Edge hub로 전달하는 웹소켓 서버입니다.
- [EdgeController](https://kubeedge.io/en/docs/architecture/cloud/edge_controller): 데이터가 특정 엣지노드의 대상이 될 수 있게 엣지 노드 및 포드 메타데이터를 관리하는 쿠버네티스 확장 컨트롤러 입니다.
- [DeviceController](https://kubeedge.io/en/docs/architecture/cloud/device_controller): 엣지와 클라우드 간 장치의 메타데이터/상태데이터가 동기화 될 수 있도록 장치를 관리하는 쿠버네티스 확장 컨트롤러 입니다. 


### 엣지 파트
- [EdgeHub](https://kubeedge.io/en/docs/architecture/edge/edgehub): 엣지 컴퓨팅을 위해 클라우드 서비스와 상호작용하는 웹 소켓 클라이언트(KubeEdge 아키텍쳐의 Edge Controller와 비슷함). 여기에는 클라우드 사이드의 리소스 업데이트를 엣지에 동기화하고, 엣지 사이드 호스트 및 장치 상태 변화를 클라우드에 보고하는 기능이 포함되어 있음. 
- [Edged](https://kubeedge.io/en/docs/architecture/edge/edged): 엣지노드 에서 실행되는 에이전트 프로그램으로 컨테이너화된 어플리케이션을 관리합니다.
- [EventBus](https://kubeedge.io/en/docs/architecture/edge/eventbus): MQTT 서버(mosquitto)와 상호 작용하는 MQTT 클라이언트로 다른 구성 요소에 대한 게시 및 구독 기능을 제공합니다.
- [ServiceBus](https://kubeedge.io/en/docs/architecture/edge/servicebus): HTTP REST 서버 와 상호작용하는 HTTP 클라이언트, 클라우드 구성 요소에 대한 HTTP 클라이언트 기능을 통해서 엣지에서 실행되는 HTTP 서버에 접근 할 수 있게 합니다.
- [DeviceTwin](https://kubeedge.io/en/docs/architecture/edge/devicetwin): 장치 상태를 저장하고 장치 상태를 클라우드에 동기화하는 일을 담당합니다. 또한 어플리케이션에 대한 쿼리 인터페이스를 제공합니다.
- [MetaManager](https://kubeedge.io/en/docs/architecture/edge/metamanager): edged와 edgehub 사이의 메시지 처리기. 메타 데이터를 경량 데이터베이스(SQLite)에 저장 및 조회하는 기능도 담당한다.

## 쿠버네티스 호환

|                        | Kubernetes 1.16 | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20 | Kubernetes 1.21 | Kubernetes 1.22 |
|------------------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|-----------------|
| KubeEdge 1.10          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge 1.11          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge 1.12          | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |
| KubeEdge HEAD (master) | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               | ✓               |

Key:
* `✓` KubeEdge와 Kubernetes가 정확하게 호환됩니다.
* `+` KubeEdge에 Kubernetes 버전에 사용할 수 없는 기능 또는 API 객체가 있습니다.
* `-` Kubernetes 버전에 KubeEdge 가 사용할 수 없는 기능 또는 API 객체가 있습니다.

## 가이드

시작하려면 다음 문서를 참고하세요 [doc](https://kubeedge.io/en/docs).

좀 더 상세한 정보를 알고 싶다면 다음 문서를 참고하세요 [kubeedge.io](https://kubeedge.io).

KubeEdge를 좀 더 깊히 배우고 싶다면, 다음 링크의 몇 몇 예시들을 시도해보세요 [examples](https://github.com/kubeedge/examples).

## 로드맵

* [2021 로드맵](./docs/roadmap.md#roadmap)

## 미팅

정기 커뮤니티 미팅:
- 유럽 시간: **수요일 16:30-17:30 북경 시간 기준** (2020년 2월 19일 부터 시작해서, 격주로 진행).
([시간대를 변환하십시오.](https://www.thetimezoneconverter.com/?t=16%3A30&tz=GMT%2B8&))
- 태평양 시간: **수요일 10:00-11:00 북경 시간 기준** (2020년 2월 26일 부터 시작해서, 격주로 진행).
([시간대를 변환하십시오.](https://www.thetimezoneconverter.com/?t=10%3A00&tz=GMT%2B8&))

자료:
- [회의록 및 의제](https://docs.google.com/document/d/1Sr5QS_Z04uPfRbA7PrXr3aPwCRpx7EtsyHq7mp6CnHs/edit)
- [회의 녹음](https://www.youtube.com/playlist?list=PLQtlO1kVWGXkRGkjSrLGEPJODoPb8s5FM)
- [회의 링크](https://zoom.us/j/4167237304)
- [회의 캘린더](https://calendar.google.com/calendar/embed?src=8rjk8o516vfte21qibvlae3lj4%40group.calendar.google.com) | [구독](https://calendar.google.com/calendar?cid=OHJqazhvNTE2dmZ0ZTIxcWlidmxhZTNsajRAZ3JvdXAuY2FsZW5kYXIuZ29vZ2xlLmNvbQ)

## 연락처

지원이 필요한 경우 [문제 해결 가이드](https://kubeedge.io/en/docs/developer/troubleshooting)로 시작하여 설명된 프로세스를 진행하세요.

궁금한 점이 있으면 다음 방법으로 언제든지 문의해 주세요.

- [메일링 목록](https://groups.google.com/forum/#!forum/kubeedge)
- [슬랙](https://join.slack.com/t/kubeedge/shared_invite/enQtNjc0MTg2NTg2MTk0LWJmOTBmOGRkZWNhMTVkNGU1ZjkwNDY4MTY4YTAwNDAyMjRkMjdlMjIzYmMxODY1NGZjYzc4MWM5YmIxZjU1ZDI)
- [트위터](https://twitter.com/kubeedge)

## 컨트리뷰터

컨트리뷰터가 되는 데 관심이 있고 참여하고 싶다면, KubeEdge 코드를 개발하려면 [CONTRIBUTING](./CONTRIBUTING.md)을 참조하십시오.
패치 제출 및 컨트리뷰션 워크플로우에 대한 세부 정보.

## 보안

### 보안 감사

KubeEdge의 제3자 보안 감사가 2022년 7월에 완료되었습니다. 또한 KubeEdge 커뮤니티는 KubeEdge의 전체 시스템 보안 분석을 완료했습니다. 자세한 보고서는 다음과 같습니다.

- [보안 감사](https://github.com/kubeedge/community/blob/master/sig-security/sig-security-audit/KubeEdge-security-audit-2022.pdf)

- [위협 모델 및 보안 보호 분석 문서](https://github.com/kubeedge/community/blob/master/sig-security/sig-security-audit/KubeEdge-threat-model-and-security-protection-analysis.md)

### 보안 취약점 보고

보안 연구원, 산업 조직 및 사용자 분들은 의심되는 취약점을 발견하신다면 우리 보안 팀(`cncf-kubeedge-security@lists.cncf.io`)에 문의해주시길 바랍니다. 팀은 문제의 심각성을 진단하고 해결 방법을 결정하는 데 도움을 줄 것입니다. 발견 즉시 문의 해주시면 감사하겠습니다.

보안 프로세스 및 취약점 보고 방법에 대한 자세한 내용은 [보안 정책](https://github.com/kubeedge/community/blob/master/team-security/SECURITY.md)을 참조하세요.

## 라이센스

KubeEdge는 Apache 2.0 라이선스를 따릅니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하십시오.
