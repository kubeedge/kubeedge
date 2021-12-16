module github.com/kubeedge/kubeedge

go 1.16

require (
	github.com/256dpi/gomqtt v0.10.4
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/astaxie/beego v1.12.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/container-storage-interface/spec v1.3.0
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/docker/docker v20.10.2+incompatible
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/cadvisor v0.39.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/kubeedge/beehive v0.0.0
	github.com/kubeedge/viaduct v0.0.0
	github.com/kubernetes-csi/csi-lib-utils v0.6.1
	github.com/lib/pq v1.2.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/paypal/gatt v0.0.0-20151011220935-4ae819d591cf
	github.com/prometheus/client_golang v1.7.1
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	google.golang.org/grpc v1.27.1
	k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/apiserver v0.21.4
	k8s.io/cli-runtime v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/cloud-provider v0.21.4
	k8s.io/cluster-bootstrap v0.21.4 // indirect
	k8s.io/code-generator v0.21.4
	k8s.io/component-base v0.21.4
	k8s.io/cri-api v0.21.4
	k8s.io/csi-translation-lib v0.21.4
	k8s.io/klog/v2 v2.8.0
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	k8s.io/kube-scheduler v0.21.4 // indirect
	k8s.io/kubelet v0.21.4
	k8s.io/kubernetes v1.21.4
	k8s.io/mount-utils v0.21.4
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/apiserver-network-proxy v0.0.20
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.22
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Sirupsen/logrus v1.0.5 => github.com/sirupsen/logrus v1.0.5
	github.com/Sirupsen/logrus v1.3.0 => github.com/Sirupsen/logrus v1.0.6
	github.com/Sirupsen/logrus v1.4.0 => github.com/sirupsen/logrus v1.0.6
	github.com/gopherjs/gopherjs v0.0.0 => github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/kubeedge/beehive => ./staging/src/github.com/kubeedge/beehive
	github.com/kubeedge/viaduct => ./staging/src/github.com/kubeedge/viaduct
	k8s.io/api v0.0.0 => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.21.4
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.21.4
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.21.4
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.21.4
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.21.4
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.21.4
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.21.4
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.21.4
	k8s.io/component-helpers v0.0.0 => k8s.io/component-helpers v0.21.4
	k8s.io/controller-manager v0.0.0 => k8s.io/controller-manager v0.21.4
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.21.4
	k8s.io/csi-api v0.0.0 => k8s.io/csi-api v0.21.4
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.21.4
	k8s.io/gengo v0.0.0 => k8s.io/gengo v0.21.4
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1 // indirect
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.8.0
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.21.4
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.21.4
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.21.4
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.21.4
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.21.4
	k8s.io/kubectl => k8s.io/kubectl v0.21.4
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.21.4
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.21.4
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.21.4
	k8s.io/mount-utils v0.0.0 => k8s.io/mount-utils v0.21.4
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.21.4
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.21.4
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.21.4
	k8s.io/utils v0.0.0 => k8s.io/utils v0.21.4
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client => sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.22
)
