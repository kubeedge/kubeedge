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
	github.com/goburrow/serial v0.1.0 // indirect
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/cadvisor v0.37.0
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
	github.com/sailorvii/modbus v0.1.2
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	google.golang.org/grpc v1.27.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/apiserver v0.19.3
	k8s.io/cli-runtime v0.19.1
	k8s.io/client-go v0.19.3
	k8s.io/cloud-provider v0.19.3
	k8s.io/cluster-bootstrap v0.19.3 // indirect
	k8s.io/code-generator v0.19.3
	k8s.io/component-base v0.19.3
	k8s.io/cri-api v0.19.3
	k8s.io/csi-translation-lib v0.19.3
	k8s.io/klog/v2 v2.2.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kube-scheduler v0.19.3 // indirect
	k8s.io/kubelet v0.19.3
	k8s.io/kubernetes v1.19.3
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
	k8s.io/api v0.0.0 => k8s.io/api v0.0.0-20190720062849-3043179095b6
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.0.0-20190718185103-d1ef975d28ce // indirect
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.0.0-20190718184206-a1aa83af71a7
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.0.0-20190718185405-0ce9869d0015
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.0.0-20190718183610-8e956561bbf5 // indirect
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.0.0-20190718190308-f8e43aa19282 // indirect
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.0.0-20190718190146-f7b0473036f9
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.15.8-beta.1
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.0.0-20190718183727-0ececfbe9772
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.0.0-20190531030430-6117653b35f1
	k8s.io/csi-api v0.0.0 => k8s.io/csi-api v0.0.0-20190313123203-94ac839bf26c // indirect
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.0.0-20190718190424-bef8d46b95de
	k8s.io/gengo v0.0.0 => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1 // indirect
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.2.0
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.0.0-20190718184434-a064d4d1ed7a
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.0.0-20190718190030-ea930fedc880
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.0.0-20190718094010-3cf2ea392886 // indirect
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.0.0-20190718185641-5233cb7cb41e
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.0.0-20190718185913-d5429d807831
	k8s.io/kubectl => k8s.io/kubectl v0.19.1
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.0.0-20190718185757-9b45f80d5747
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.0.0-20190718190548-039b99e58dbd
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.0.0-20190718185242-1e1642704fe6
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.0.0-20190717025432-9e6fdeee55cc // indirect
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.0.0-20181204233714-00fe14e3d1a3 // indirect
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.0.0-20190718184639-baafa86838c0
	k8s.io/utils v0.0.0 => k8s.io/utils v0.0.0-20190712204705-3dccf664f023
)
