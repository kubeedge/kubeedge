module github.com/kubeedge/viaduct

go 1.17

require (
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.4.2
	github.com/kubeedge/beehive v0.0.0
	github.com/lucas-clemente/quic-go v0.10.1
	k8s.io/klog/v2 v2.9.0
)

require (
	github.com/bifurcation/mint v0.0.0-20180715133206-93c51c6ce115 // indirect
	github.com/cheekybits/genny v0.0.0-20170328200008-9127e812e1e9 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.0.0-20180201235237-0fb14efe8c47 // indirect
	github.com/lucas-clemente/aes12 v0.0.0-20171027163421-cd47fb39b79f // indirect
	github.com/lucas-clemente/quic-go-certificates v0.0.0-20160823095156-d2f86524cced // indirect
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
)

replace (
	github.com/kubeedge/beehive => ../beehive
	github.com/kubeedge/viaduct => ../viaduct
)
