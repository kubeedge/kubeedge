module github.com/kubeedge/kubeedge

go 1.13

require (
	cloud.google.com/go v0.43.0 // indirect
	github.com/256dpi/gomqtt v0.10.4
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/astaxie/beego v1.12.0
	github.com/aws/aws-sdk-go v1.21.10 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/container-storage-interface/spec v1.2.0
	github.com/containerd/console v0.0.0-20181022165439-0650fd9eeb50 // indirect
	github.com/containerd/containerd v1.1.7 // indirect
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-chassis/go-chassis v1.7.1
	github.com/go-chassis/paas-lager v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/cadvisor v0.35.0
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/karrick/godirwalk v1.10.12 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kubeedge/beehive v0.0.0
	github.com/kubeedge/viaduct v0.0.0
	github.com/kubernetes-csi/csi-lib-utils v0.6.1
	github.com/mattn/go-sqlite3 v1.11.0
	github.com/mesos/mesos-go v0.0.10 // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/opencontainers/runtime-spec v1.0.1 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/paypal/gatt v0.0.0-20151011220935-4ae819d591cf
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/sys v0.0.0-20190826190057-c7b8b68b1456
	google.golang.org/grpc v1.23.1
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.1
	k8s.io/apiextensions-apiserver v0.17.1
	k8s.io/apimachinery v0.17.1
	k8s.io/apiserver v0.17.1
	k8s.io/client-go v0.17.1
	k8s.io/cloud-provider v0.17.1
	k8s.io/code-generator v0.17.1
	k8s.io/component-base v0.17.1
	k8s.io/cri-api v0.17.1
	k8s.io/csi-translation-lib v0.17.1
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kubelet v0.17.1
	k8s.io/kubernetes v1.17.1
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/yaml v1.1.0
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
	k8s.io/klog => k8s.io/klog v0.4.0 // indirect
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.0.0-20190718184434-a064d4d1ed7a
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.0.0-20190718190030-ea930fedc880
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.0.0-20190718094010-3cf2ea392886 // indirect
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.0.0-20190718185641-5233cb7cb41e
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.0.0-20190718185913-d5429d807831
	k8s.io/kubectl => k8s.io/kubectl v0.17.0
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.0.0-20190718185757-9b45f80d5747
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.0.0-20190718190548-039b99e58dbd
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.0.0-20190718185242-1e1642704fe6
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.0.0-20190717025432-9e6fdeee55cc // indirect
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.0.0-20181204233714-00fe14e3d1a3 // indirect
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.0.0-20190718184639-baafa86838c0
	k8s.io/utils v0.0.0 => k8s.io/utils v0.0.0-20190712204705-3dccf664f023
)
