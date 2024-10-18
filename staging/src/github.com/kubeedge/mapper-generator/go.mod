module github.com/kubeedge/mapper-generator

go 1.17

require (
	github.com/fatih/structs v1.1.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/kubeedge/kubeedge v1.12.0-beta.0
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.80.1
)

require (
	github.com/go-logr/logr v1.2.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
)

replace (
	github.com/kubeedge/beehive v0.0.0 => github.com/kubeedge/beehive v1.7.0
	github.com/kubeedge/viaduct v0.0.0 => github.com/kubeedge/viaduct v0.0.0-20201130063818-e33931917980
	k8s.io/api v0.0.0 => k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.19.3
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.19.3
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.19.3
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.19.3
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.19.3
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.19.3
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.19.3
	k8s.io/component-helpers v0.0.0 => k8s.io/component-helpers v0.22.6
	k8s.io/controller-manager v0.0.0 => k8s.io/controller-manager v0.22.6
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.19.3
	k8s.io/csi-api v0.0.0 => k8s.io/csi-api v0.19.3
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.19.3
	k8s.io/gengo v0.0.0 => k8s.io/gengo v0.19.3
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1 // indirect
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.19.3
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.19.3
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.19.3
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.19.3
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.19.3
	k8s.io/kubectl => k8s.io/kubectl v0.19.3
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.19.3
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.19.3
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.19.3
	k8s.io/mount-utils v0.0.0 => k8s.io/mount-utils v0.22.6
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.19.3
	k8s.io/pod-security-admission v0.0.0 => k8s.io/pod-security-admission v0.22.6
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.19.3
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.19.3
	k8s.io/utils v0.0.0 => k8s.io/utils v0.19.3
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.0 => sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.27
)
