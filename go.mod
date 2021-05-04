module github.com/kubeedge/kubeedge

go 1.14

require (
	github.com/256dpi/gomqtt v0.10.4
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/astaxie/beego v1.12.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/container-storage-interface/spec v1.2.0
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/docker v1.4.2-0.20200309214505-aa6a9891b09c
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-chassis/go-archaius v0.20.0
	github.com/go-chassis/go-chassis v1.7.1
	github.com/go-chassis/paas-lager v1.1.1 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/cadvisor v0.37.5
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/karrick/godirwalk v1.10.12 // indirect
	github.com/kubeedge/beehive v0.0.0
	github.com/kubeedge/viaduct v0.0.0
	github.com/kubernetes-csi/csi-lib-utils v0.6.1
	github.com/lib/pq v1.2.0 // indirect
	github.com/mattn/go-sqlite3 v1.11.0
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/paypal/gatt v0.0.0-20151011220935-4ae819d591cf
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	google.golang.org/grpc v1.27.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.19.10
	k8s.io/apiextensions-apiserver v0.19.10
	k8s.io/apimachinery v0.19.10
	k8s.io/apiserver v0.19.10
	k8s.io/cli-runtime v0.19.10
	k8s.io/client-go v0.19.10
	k8s.io/cloud-provider v0.19.10
	k8s.io/cluster-bootstrap v0.19.10 // indirect
	k8s.io/code-generator v0.19.10
	k8s.io/component-base v0.19.10
	k8s.io/cri-api v0.19.10
	k8s.io/csi-translation-lib v0.19.10
	k8s.io/klog/v2 v2.2.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kube-scheduler v0.19.10 // indirect
	k8s.io/kubelet v0.19.10
	k8s.io/kubernetes v1.19.10
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Sirupsen/logrus v1.0.5 => github.com/sirupsen/logrus v1.0.5
	github.com/Sirupsen/logrus v1.3.0 => github.com/Sirupsen/logrus v1.0.6
	github.com/Sirupsen/logrus v1.4.0 => github.com/sirupsen/logrus v1.0.6
	github.com/apache/servicecomb-kie v0.1.0 => github.com/apache/servicecomb-kie v0.0.0-20190905062319-5ee098c8886f // indirect. TODO: remove this line when servicecomb-kie has a stable release
	github.com/gopherjs/gopherjs v0.0.0 => github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/kubeedge/beehive => ./staging/src/github.com/kubeedge/beehive
	github.com/kubeedge/viaduct => ./staging/src/github.com/kubeedge/viaduct
	k8s.io/api v0.0.0 => k8s.io/api v0.19.10
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.19.10
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.19.10
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.19.10
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.19.10
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.19.10
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.19.10
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.19.10
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.19.10
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.19.10
	k8s.io/csi-api v0.0.0 => k8s.io/csi-api v0.19.10
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.19.10
	k8s.io/gengo v0.0.0 => k8s.io/gengo v0.19.10
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1 // indirect
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.2.0
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.19.10
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.19.10
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.19.10
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.19.10
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.19.10
	k8s.io/kubectl => k8s.io/kubectl v0.19.10
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.19.10
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.19.10
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.19.10
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.19.10
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.19.10
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.19.10
	k8s.io/utils v0.0.0 => k8s.io/utils v0.19.10
)
